package s3manager

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestSortAndPaginateObjects(t *testing.T) {
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

		objs, totalItems, totalPages, page := sortAndPaginateObjects(versionedObjs(), "size", "asc", 1, 25, false, true)

		is.Equal(2, totalItems) // groups, not versions
		is.Equal(1, totalPages)
		is.Equal(1, page)
		is.Equal(3, len(objs))
		// b.txt (primary size 50) sorts before the a.txt group (primary size 100),
		// and a.txt keeps its listing order (newest version first).
		is.Equal("b.txt", objs[0].Key)
		is.Equal("v2", objs[1].VersionID)
		is.Equal("v1", objs[2].VersionID)
	})

	t.Run("never splits a version group across pages", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		objs, totalItems, totalPages, page := sortAndPaginateObjects(versionedObjs(), "key", "asc", 1, 1, false, true)

		is.Equal(2, totalItems)
		is.Equal(2, totalPages)
		is.Equal(1, page)
		// Page 1 holds the whole a.txt group.
		is.Equal(2, len(objs))
		is.Equal("a.txt", objs[0].Key)
		is.Equal("a.txt", objs[1].Key)

		objs, _, _, page = sortAndPaginateObjects(versionedObjs(), "key", "asc", 2, 1, false, true)
		is.Equal(2, page)
		is.Equal(1, len(objs))
		is.Equal("b.txt", objs[0].Key)
	})

	t.Run("clamps the page number to the last page", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		objs, _, totalPages, page := sortAndPaginateObjects(versionedObjs(), "key", "asc", 99, 1, false, true)

		is.Equal(2, totalPages)
		is.Equal(2, page)
		is.Equal("b.txt", objs[0].Key)
	})

	t.Run("returns everything when showAll is set", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		objs, totalItems, totalPages, page := sortAndPaginateObjects(versionedObjs(), "key", "asc", 3, 1, true, true)

		is.Equal(2, totalItems)
		is.Equal(1, totalPages)
		is.Equal(1, page)
		is.Equal(3, len(objs))
	})

	t.Run("sorts flat when grouping is disabled", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		now := time.Now()
		objs := []objectWithIcon{
			{Key: "b.txt", DisplayName: "b.txt", LastModified: now},
			{Key: "a.txt", DisplayName: "a.txt", LastModified: now.Add(time.Hour)},
		}

		sorted, totalItems, totalPages, page := sortAndPaginateObjects(objs, "lastModified", "desc", 1, 25, false, false)

		is.Equal(2, totalItems)
		is.Equal(1, totalPages)
		is.Equal(1, page)
		is.Equal("a.txt", sorted[0].Key)
		is.Equal("b.txt", sorted[1].Key)
	})
}
