package s3manager

import (
	"context"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// objectWithIcon is an S3 object as shown in the bucket view.
type objectWithIcon struct {
	Key              string
	Size             int64
	SizeDisplay      string
	LastModified     time.Time
	Owner            string
	Icon             string
	IsFolder         bool
	DisplayName      string
	VersionID        string
	IsLatest         bool
	IsDeleteMarker   bool
	VersionCount     int
	GroupIndex       int
	IsPrimaryVersion bool
}

// listObjects lists a bucket's objects, converting each minio.ObjectInfo into an
// objectWithIcon. If showVersions is set but the versioned listing fails, or
// comes back empty (some S3-compatible providers don't support listing object
// versions and either reject the request outright or silently return nothing
// instead of erroring), it transparently falls back to a normal listing so the
// bucket can still be browsed. The returned bool reports whether version
// information is actually present in the result.
func listObjects(ctx context.Context, s3 S3, bucketName, prefix string, listRecursive, showVersions bool) ([]objectWithIcon, bool, error) {
	opts := minio.ListObjectsOptions{Recursive: listRecursive, Prefix: prefix}

	if showVersions {
		opts.WithVersions = true
		objs, err := collectObjects(ctx, s3, bucketName, prefix, opts)
		if err == nil && len(objs) > 0 {
			return objs, true, nil
		}
		opts.WithVersions = false
	}

	objs, err := collectObjects(ctx, s3, bucketName, prefix, opts)
	if err != nil {
		return nil, false, err
	}

	return objs, false, nil
}

// collectObjects drains an S3 ListObjects channel into a slice, returning the
// first error encountered (if any) instead of a partial, half-listed result.
func collectObjects(ctx context.Context, s3 S3, bucketName, prefix string, opts minio.ListObjectsOptions) ([]objectWithIcon, error) {
	var objs []objectWithIcon
	for object := range s3.ListObjects(ctx, bucketName, opts) {
		if object.Err != nil {
			return nil, object.Err
		}
		objs = append(objs, toObjectWithIcon(object, prefix))
	}

	return objs, nil
}

// toObjectWithIcon converts a minio.ObjectInfo into the template-facing objectWithIcon.
func toObjectWithIcon(object minio.ObjectInfo, prefix string) objectWithIcon {
	return objectWithIcon{
		Key:            object.Key,
		Size:           object.Size,
		SizeDisplay:    formatFileSize(object.Size),
		LastModified:   object.LastModified,
		Owner:          object.Owner.DisplayName,
		Icon:           icon(object.Key),
		IsFolder:       strings.HasSuffix(object.Key, "/"),
		DisplayName:    strings.TrimSuffix(strings.TrimPrefix(object.Key, prefix), "/"),
		VersionID:      object.VersionID,
		IsLatest:       object.IsLatest,
		IsDeleteMarker: object.IsDeleteMarker,
	}
}

// icon returns an icon for a file type.
func icon(fileName string) string {
	if strings.HasSuffix(fileName, "/") {
		return "folder"
	}

	switch path.Ext(fileName) {
	case ".tgz", ".gz", ".zip":
		return "archive"
	case ".png", ".jpg", ".gif", ".svg":
		return "photo"
	case ".mp3", ".wav":
		return "music_note"
	default:
		return "insert_drive_file"
	}
}

// annotateVersionGroups sets VersionCount, GroupIndex and IsPrimaryVersion on
// each object so the template can collapse older versions under their latest
// version by default. IsPrimaryVersion picks exactly one visible row per key:
// the one the provider marked IsLatest, or (since some S3-compatible providers
// leave IsLatest unset — notably folder entries synthesized from
// CommonPrefixes, which are never version-aware) the first entry seen for that
// key. Relying on the raw IsLatest flag alone would hide every row in a group
// where no entry has it set, making the bucket appear empty.
func annotateVersionGroups(objs []objectWithIcon) {
	counts := make(map[string]int, len(objs))
	groupIndex := make(map[string]int, len(objs))
	primaryIndex := make(map[string]int, len(objs))

	for i, obj := range objs {
		counts[obj.Key]++
		if _, ok := groupIndex[obj.Key]; !ok {
			groupIndex[obj.Key] = len(groupIndex)
			primaryIndex[obj.Key] = i
		} else if obj.IsLatest {
			primaryIndex[obj.Key] = i
		}
	}

	for i := range objs {
		key := objs[i].Key
		objs[i].VersionCount = counts[key]
		objs[i].GroupIndex = groupIndex[key]
		objs[i].IsPrimaryVersion = primaryIndex[key] == i
	}
}

// filterObjects keeps the objects whose key or display name contains the
// case-insensitive search term.
func filterObjects(objs []objectWithIcon, search string) []objectWithIcon {
	search = strings.ToLower(search)

	filtered := make([]objectWithIcon, 0, len(objs))
	for _, obj := range objs {
		if strings.Contains(strings.ToLower(obj.DisplayName), search) ||
			strings.Contains(strings.ToLower(obj.Key), search) {
			filtered = append(filtered, obj)
		}
	}

	return filtered
}

// objectPage is the slice of a bucket's objects shown on a single page.
type objectPage struct {
	Objects    []objectWithIcon
	Page       int
	PerPage    int
	TotalItems int
	TotalPages int
}

// HasPrevPage reports whether there is a page before the current one.
func (p objectPage) HasPrevPage() bool { return p.Page > 1 }

// HasNextPage reports whether there is a page after the current one.
func (p objectPage) HasNextPage() bool { return p.Page < p.TotalPages }

// paginateObjects sorts objects and slices out the requested page. When grouped
// is set (versioned listing), all versions of a key travel together: groups are
// ordered by their primary row and are never split across page boundaries, and
// TotalItems counts objects, not individual versions.
func paginateObjects(objs []objectWithIcon, query listingQuery, grouped bool) objectPage {
	groups := groupObjects(objs, grouped)
	sortObjectGroups(groups, query.SortBy, query.SortOrder)

	totalItems := len(groups)
	if query.ShowAll {
		return objectPage{
			Objects: flattenGroups(groups),
			Page:    1,
			// The template shows a page range, which needs a non-zero size.
			PerPage:    max(totalItems, 1),
			TotalItems: totalItems,
			TotalPages: 1,
		}
	}

	perPage := max(query.PerPage, 1)
	totalPages := max((totalItems+perPage-1)/perPage, 1)
	page := min(query.Page, totalPages)
	start := min((page-1)*perPage, totalItems)
	end := min(start+perPage, totalItems)

	return objectPage{
		Objects:    flattenGroups(groups[start:end]),
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

// groupObjects splits objs into version groups that move as one unit through
// sorting and pagination. Objects keep their listing order within a group.
// Without version grouping every object is its own group.
func groupObjects(objs []objectWithIcon, grouped bool) [][]objectWithIcon {
	if !grouped {
		groups := make([][]objectWithIcon, len(objs))
		for i := range objs {
			groups[i] = objs[i : i+1 : i+1]
		}
		return groups
	}

	positions := make(map[int]int, len(objs))
	var groups [][]objectWithIcon
	for _, obj := range objs {
		pos, ok := positions[obj.GroupIndex]
		if !ok {
			pos = len(groups)
			positions[obj.GroupIndex] = pos
			groups = append(groups, nil)
		}
		groups[pos] = append(groups[pos], obj)
	}

	return groups
}

// sortObjectGroups sorts version groups based on the specified field and order,
// comparing groups by their primary row so all versions of a key move as one
// unit. The stable sort preserves the S3 listing order between equal groups.
func sortObjectGroups(groups [][]objectWithIcon, sortBy, sortOrder string) {
	sort.SliceStable(groups, func(i, j int) bool {
		a := primaryObject(groups[i])
		b := primaryObject(groups[j])

		var less bool
		switch sortBy {
		case "size":
			less = a.Size < b.Size
		case "owner":
			less = strings.ToLower(a.Owner) < strings.ToLower(b.Owner)
		case "lastModified":
			less = a.LastModified.Before(b.LastModified)
		default:
			less = strings.ToLower(a.DisplayName) < strings.ToLower(b.DisplayName)
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

// primaryObject returns the row that represents a group when sorting: the one
// marked IsPrimaryVersion by annotateVersionGroups, or the first row otherwise.
func primaryObject(group []objectWithIcon) objectWithIcon {
	for _, obj := range group {
		if obj.IsPrimaryVersion {
			return obj
		}
	}

	return group[0]
}

// flattenGroups concatenates version groups back into a flat object list.
func flattenGroups(groups [][]objectWithIcon) []objectWithIcon {
	objs := make([]objectWithIcon, 0, len(groups))
	for _, group := range groups {
		objs = append(objs, group...)
	}

	return objs
}
