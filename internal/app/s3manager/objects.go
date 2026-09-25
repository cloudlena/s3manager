package s3manager

import (
	"context"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// listedObject is an S3 object as shown in the bucket view.
type listedObject struct {
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

// maxScanObjects bounds how many objects a single full listing pulls out of S3.
// Sorting by another column than the key, sorting descending, searching, an
// exact object count and version grouping all need the whole prefix in memory,
// so those requests still scan it — but a bucket with millions of keys would
// otherwise tie up the request indefinitely. A listing that hits this cap is
// reported as truncated so the view can say that its results are partial.
const maxScanObjects = 10000

// endOfFolder sorts after every object a folder realistically contains, so
// appending it to a folder's key yields a cursor that resumes the listing past
// the folder's contents. It is the highest Unicode code point there is; a key
// above it would only make the folder show up on the next page again.
const endOfFolder = "\U0010FFFF"

// objectListing is the outcome of listing a bucket prefix for the bucket view.
type objectListing struct {
	// Objects are the objects that were listed, in S3 listing order.
	Objects []listedObject
	// VersionsShown reports whether Objects carry version information.
	VersionsShown bool
	// Truncated reports that the prefix holds more objects than were listed,
	// so counting, sorting and searching only cover Objects.
	Truncated bool
	// NextCursor resumes the listing after the last object of a single page,
	// and is empty on the last page. Only set by listObjectPage.
	NextCursor string
}

// listAllObjects lists a bucket prefix in full, up to maxScanObjects, converting
// each minio.ObjectInfo into a listedObject. If showVersions is set but the
// versioned listing fails, or comes back empty (some S3-compatible providers
// don't support listing object versions and either reject the request outright
// or silently return nothing instead of erroring), it transparently falls back
// to a normal listing so the bucket can still be browsed.
func listAllObjects(ctx context.Context, s3 S3, bucketName, prefix string, listRecursive, showVersions bool) (objectListing, error) {
	opts := minio.ListObjectsOptions{Recursive: listRecursive, Prefix: prefix}

	if showVersions {
		opts.WithVersions = true
		objs, err := collectObjects(ctx, s3, bucketName, prefix, maxScanObjects+1, opts)
		if err == nil && len(objs) > 0 {
			return cappedListing(objs, true), nil
		}
		opts.WithVersions = false
	}

	objs, err := listWithV1Fallback(ctx, s3, bucketName, prefix, maxScanObjects+1, opts)
	if err != nil {
		return objectListing{}, err
	}

	return cappedListing(objs, false), nil
}

// cappedListing trims a listing that collected one object beyond maxScanObjects
// down to the cap, flagging it as truncated.
func cappedListing(objs []listedObject, versionsShown bool) objectListing {
	listing := objectListing{Objects: objs, VersionsShown: versionsShown}
	if len(objs) > maxScanObjects {
		listing.Objects = objs[:maxScanObjects]
		listing.Truncated = true
	}

	return listing
}

// listObjectPage lists a single page of a bucket prefix straight from S3, asking
// for just the objects the page shows instead of scanning the whole prefix. It
// leans on S3 listing keys in ascending lexicographic order and resuming after a
// given key, so it can only serve a key-ascending listing without search — see
// cursorPagingPossible. Pages are ordered by raw key rather than by the
// case-insensitive display name a full listing sorts by, because that is the
// only order S3 itself can paginate in.
func listObjectPage(ctx context.Context, s3 S3, bucketName, prefix, cursor string, listRecursive bool, perPage int) (objectListing, error) {
	perPage = max(perPage, 1)
	opts := minio.ListObjectsOptions{
		Recursive:  listRecursive,
		Prefix:     prefix,
		StartAfter: cursor,
		// minio's MaxKeys is the batch size, not a total. Asking for exactly
		// one object more than the page shows keeps a page at a single S3
		// round trip while still revealing whether a next page exists.
		MaxKeys: perPage + 1,
	}

	objs, err := listWithV1Fallback(ctx, s3, bucketName, prefix, perPage+1, opts)
	if err != nil {
		return objectListing{}, err
	}

	// A delimited (non-recursive) listing reports a batch's objects before its
	// folders, so the batch reaches us out of key order even though S3 produced
	// it in order.
	sortObjectsByKey(objs)

	listing := objectListing{Objects: objs}
	if len(objs) > perPage {
		listing.Objects = objs[:perPage]
		listing.NextCursor = nextCursor(listing.Objects[perPage-1])
	}

	return listing, nil
}

// nextCursor returns the key a listing has to resume after to continue directly
// below the given object. A folder is a common prefix rather than a real key and
// everything inside it sorts after it ("dir/" < "dir/file"), so resuming after
// the prefix itself would collapse the very same folder into the listing again,
// forever.
func nextCursor(obj listedObject) string {
	if obj.IsFolder {
		return obj.Key + endOfFolder
	}

	return obj.Key
}

// listWithV1Fallback lists a prefix and, when that comes back empty, lists it
// once more as a ListObjects V1 request. minio-go always speaks V2 and never
// falls back on its own, while some S3-compatible providers answer a V2 listing
// they don't implement with an empty result instead of an error — which is
// indistinguishable from an empty prefix and makes a bucket full of objects
// look empty, with nothing to report to the user. The retry costs one extra
// round trip on prefixes that really are empty, and turns that silent case into
// the objects that are actually there. StartAfter doubles as V1's marker, so a
// cursor-paged listing survives the switch. If the retry itself fails, the
// empty V2 listing stands: a provider that rejects V1 is one that meant its
// empty answer.
func listWithV1Fallback(ctx context.Context, s3 S3, bucketName, prefix string, limit int, opts minio.ListObjectsOptions) ([]listedObject, error) {
	objs, err := collectObjects(ctx, s3, bucketName, prefix, limit, opts)
	if err != nil || len(objs) > 0 {
		return objs, err
	}

	opts.UseV1 = true
	v1Objs, err := collectObjects(ctx, s3, bucketName, prefix, limit, opts)
	if err != nil {
		return objs, nil
	}

	return v1Objs, nil
}

// collectObjects drains up to limit objects off an S3 ListObjects channel,
// returning the first error encountered (if any) instead of a partial,
// half-listed result. The context handed to ListObjects is cancelled on return,
// which is what releases minio's producer goroutine when the limit cuts the
// listing short.
func collectObjects(ctx context.Context, s3 S3, bucketName, prefix string, limit int, opts minio.ListObjectsOptions) ([]listedObject, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	objs := make([]listedObject, 0, min(limit, defaultPerPage))
	for object := range s3.ListObjects(ctx, bucketName, opts) {
		if object.Err != nil {
			return nil, object.Err
		}
		objs = append(objs, toListedObject(object, prefix))
		if len(objs) == limit {
			break
		}
	}

	return objs, nil
}

// sortObjectsByKey puts objects into the ascending lexicographic key order S3
// lists them in.
func sortObjectsByKey(objs []listedObject) {
	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })
}

// toListedObject converts a minio.ObjectInfo into the template-facing listedObject.
func toListedObject(object minio.ObjectInfo, prefix string) listedObject {
	return listedObject{
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

	switch strings.ToLower(path.Ext(fileName)) {
	case ".tgz", ".gz", ".zip":
		return "archive"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg":
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
func annotateVersionGroups(objs []listedObject) {
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
func filterObjects(objs []listedObject, search string) []listedObject {
	search = strings.ToLower(search)

	filtered := make([]listedObject, 0, len(objs))
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
	Objects    []listedObject
	Page       int
	PerPage    int
	TotalItems int
	TotalPages int
	// ShowAll reports that the page holds the whole listing.
	ShowAll bool

	// CursorPaging reports that the page came straight out of S3 instead of
	// being sliced out of a full listing. TotalItems and TotalPages are unknown
	// in that case, and the pager navigates by cursor instead of page number.
	CursorPaging bool
	// Cursors is the trail of cursors of the pages before this one, which is
	// what gives a cursor-paged view its page number and its way back.
	Cursors []string
	// NextCursor is the cursor of the following page, empty on the last page.
	NextCursor string
}

// HasPrevPage reports whether there is a page before the current one.
func (p objectPage) HasPrevPage() bool { return p.Page > 1 }

// HasNextPage reports whether there is a page after the current one.
func (p objectPage) HasNextPage() bool {
	if p.CursorPaging {
		return p.NextCursor != ""
	}

	return p.Page < p.TotalPages
}

// FirstItem is the 1-based position of the page's first object within the whole
// listing, or 0 for an empty page.
func (p objectPage) FirstItem() int {
	if len(p.Objects) == 0 {
		return 0
	}

	return (p.Page-1)*p.PerPage + 1
}

// LastItem is the 1-based position of the page's last object within the whole
// listing, or 0 for an empty page.
func (p objectPage) LastItem() int {
	if len(p.Objects) == 0 {
		return 0
	}
	if p.CursorPaging {
		return p.FirstItem() + len(p.Objects) - 1
	}

	return min(p.Page*p.PerPage, p.TotalItems)
}

// PageLinks lists the page numbers the numbered pager shows: the current page
// with up to two neighbours on either side, plus the first and the last page.
// A 0 marks a gap between pages that are not adjacent.
func (p objectPage) PageLinks() []int {
	start := max(p.Page-2, 1)
	end := min(p.Page+2, p.TotalPages)

	var links []int
	if start > 1 {
		links = append(links, 1)
		if start > 2 {
			links = append(links, 0)
		}
	}
	for i := start; i <= end; i++ {
		links = append(links, i)
	}
	if end < p.TotalPages {
		if end < p.TotalPages-1 {
			links = append(links, 0)
		}
		links = append(links, p.TotalPages)
	}

	return links
}

// cursorPage presents a page listed straight from S3. Its position in the
// listing follows from the cursor trail: every page before it was full, so the
// offset is exact even though the total number of objects is unknown.
func cursorPage(listing objectListing, query listingQuery) objectPage {
	return objectPage{
		Objects:      listing.Objects,
		Page:         len(query.Cursors) + 1,
		PerPage:      max(query.PerPage, 1),
		CursorPaging: true,
		Cursors:      query.Cursors,
		NextCursor:   listing.NextCursor,
	}
}

// paginateObjects sorts objects and slices out the requested page. When grouped
// is set (versioned listing), all versions of a key travel together: groups are
// ordered by their primary row and are never split across page boundaries, and
// TotalItems counts objects, not individual versions.
func paginateObjects(objs []listedObject, query listingQuery, grouped bool) objectPage {
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
			ShowAll:    true,
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
func groupObjects(objs []listedObject, grouped bool) [][]listedObject {
	if !grouped {
		groups := make([][]listedObject, len(objs))
		for i := range objs {
			groups[i] = objs[i : i+1 : i+1]
		}
		return groups
	}

	positions := make(map[int]int, len(objs))
	var groups [][]listedObject
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
func sortObjectGroups(groups [][]listedObject, sortBy, sortOrder string) {
	sort.SliceStable(groups, func(i, j int) bool {
		a := primaryObject(groups[i])
		b := primaryObject(groups[j])
		if sortOrder == "desc" {
			a, b = b, a
		}

		switch sortBy {
		case "size":
			return a.Size < b.Size
		case "owner":
			return strings.ToLower(a.Owner) < strings.ToLower(b.Owner)
		case "lastModified":
			return a.LastModified.Before(b.LastModified)
		default:
			return strings.ToLower(a.DisplayName) < strings.ToLower(b.DisplayName)
		}
	})
}

// primaryObject returns the row that represents a group when sorting: the one
// marked IsPrimaryVersion by annotateVersionGroups, or the first row otherwise.
func primaryObject(group []listedObject) listedObject {
	for _, obj := range group {
		if obj.IsPrimaryVersion {
			return obj
		}
	}

	return group[0]
}

// flattenGroups concatenates version groups back into a flat object list.
func flattenGroups(groups [][]listedObject) []listedObject {
	objs := make([]listedObject, 0, len(groups))
	for _, group := range groups {
		objs = append(objs, group...)
	}

	return objs
}
