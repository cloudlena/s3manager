package s3manager

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestPaginateObjects(t *testing.T) {
	t.Parallel()

	// Two versions of a.txt (latest first, as S3 lists them) and one b.txt.
	// Sorted flat by size this would interleave to a(1), b(50), a(100).
	versionedObjs := func() []objectWithIcon {
		objs := []objectWithIcon{
			{Key: "a.txt", DisplayName: "a.txt", VersionID: "v2", IsLatest: true, Size: 100},
			{Key: "a.txt", DisplayName: "a.txt", VersionID: "v1", Size: 1},
			{Key: "b.txt", DisplayName: "b.txt", VersionID: "v1", IsLatest: true, Size: 50},
		}
		annotateVersionGroups(objs)
		return objs
	}

	t.Run("keeps versions adjacent when sorting by size", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		page := paginateObjects(versionedObjs(), listingQuery{SortBy: "size", SortOrder: "asc", Page: 1, PerPage: 25}, true)

		is.Equal(2, page.TotalItems) // groups, not versions
		is.Equal(1, page.TotalPages)
		is.Equal(1, page.Page)
		is.Equal(3, len(page.Objects))
		// b.txt (primary size 50) sorts before the a.txt group (primary size 100),
		// and a.txt keeps its listing order (newest version first).
		is.Equal("b.txt", page.Objects[0].Key)
		is.Equal("v2", page.Objects[1].VersionID)
		is.Equal("v1", page.Objects[2].VersionID)
	})

	t.Run("never splits a version group across pages", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		page := paginateObjects(versionedObjs(), listingQuery{SortBy: "key", SortOrder: "asc", Page: 1, PerPage: 1}, true)

		is.Equal(2, page.TotalItems)
		is.Equal(2, page.TotalPages)
		is.Equal(1, page.Page)
		is.True(page.HasNextPage())
		is.True(!page.HasPrevPage())
		// Page 1 holds the whole a.txt group.
		is.Equal(2, len(page.Objects))
		is.Equal("a.txt", page.Objects[0].Key)
		is.Equal("a.txt", page.Objects[1].Key)

		page = paginateObjects(versionedObjs(), listingQuery{SortBy: "key", SortOrder: "asc", Page: 2, PerPage: 1}, true)
		is.Equal(2, page.Page)
		is.True(page.HasPrevPage())
		is.True(!page.HasNextPage())
		is.Equal(1, len(page.Objects))
		is.Equal("b.txt", page.Objects[0].Key)
	})

	t.Run("clamps the page number to the last page", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		page := paginateObjects(versionedObjs(), listingQuery{SortBy: "key", SortOrder: "asc", Page: 99, PerPage: 1}, true)

		is.Equal(2, page.TotalPages)
		is.Equal(2, page.Page)
		is.Equal("b.txt", page.Objects[0].Key)
	})

	t.Run("returns everything when ShowAll is set", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		page := paginateObjects(versionedObjs(), listingQuery{SortBy: "key", SortOrder: "asc", Page: 3, PerPage: 1, ShowAll: true}, true)

		is.Equal(2, page.TotalItems)
		is.Equal(1, page.TotalPages)
		is.Equal(1, page.Page)
		is.Equal(2, page.PerPage)
		is.Equal(3, len(page.Objects))
	})

	t.Run("sorts flat when grouping is disabled", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		now := time.Now()
		objs := []objectWithIcon{
			{Key: "b.txt", DisplayName: "b.txt", LastModified: now},
			{Key: "a.txt", DisplayName: "a.txt", LastModified: now.Add(time.Hour)},
		}

		page := paginateObjects(objs, listingQuery{SortBy: "lastModified", SortOrder: "desc", Page: 1, PerPage: 25}, false)

		is.Equal(2, page.TotalItems)
		is.Equal(1, page.TotalPages)
		is.Equal(1, page.Page)
		is.Equal("a.txt", page.Objects[0].Key)
		is.Equal("b.txt", page.Objects[1].Key)
	})
}
