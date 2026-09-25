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
	versionedObjs := func() []listedObject {
		objs := []listedObject{
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
		objs := []listedObject{
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

	t.Run("keeps the listing order of equal objects when sorting descending", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		objs := []listedObject{
			{Key: "a.txt", DisplayName: "a.txt", Size: 1},
			{Key: "b.txt", DisplayName: "b.txt", Size: 1},
			{Key: "c.txt", DisplayName: "c.txt", Size: 1},
		}

		page := paginateObjects(objs, listingQuery{SortBy: "size", SortOrder: "desc", Page: 1, PerPage: 25}, false)

		is.Equal("a.txt", page.Objects[0].Key)
		is.Equal("b.txt", page.Objects[1].Key)
		is.Equal("c.txt", page.Objects[2].Key)
	})

	t.Run("flags a page that holds the whole listing", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		objs := make([]listedObject, 25)
		is.True(!paginateObjects(objs, listingQuery{Page: 1, PerPage: 25}, false).ShowAll)
		is.True(paginateObjects(objs, listingQuery{Page: 1, PerPage: 25, ShowAll: true}, false).ShowAll)
	})
}

func TestPageLinks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it         string
		page       int
		totalPages int
		expected   []int
	}{
		{it: "lists a single page", page: 1, totalPages: 1, expected: []int{1}},
		{it: "lists every page of a short listing", page: 3, totalPages: 5, expected: []int{1, 2, 3, 4, 5}},
		{it: "skips the pages before the neighbours", page: 9, totalPages: 10, expected: []int{1, 0, 7, 8, 9, 10}},
		{it: "skips the pages after the neighbours", page: 2, totalPages: 10, expected: []int{1, 2, 3, 4, 0, 10}},
		{it: "skips pages on both sides", page: 5, totalPages: 10, expected: []int{1, 0, 3, 4, 5, 6, 7, 0, 10}},
		{it: "leaves no gap before an adjacent first page", page: 4, totalPages: 10, expected: []int{1, 2, 3, 4, 5, 6, 0, 10}},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			is.Equal(tc.expected, objectPage{Page: tc.page, TotalPages: tc.totalPages}.PageLinks())
		})
	}
}

func TestIcon(t *testing.T) {
	t.Parallel()

	cases := []struct {
		fileName string
		expected string
	}{
		{fileName: "photos/", expected: "folder"},
		{fileName: "archive.tar.gz", expected: "archive"},
		{fileName: "image.jpg", expected: "photo"},
		{fileName: "image.jpeg", expected: "photo"},
		{fileName: "IMG_0001.JPG", expected: "photo"},
		{fileName: "song.mp3", expected: "music_note"},
		{fileName: "notes.txt", expected: "insert_drive_file"},
	}

	for _, tc := range cases {
		t.Run(tc.fileName, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			is.Equal(icon(tc.fileName), tc.expected)
		})
	}
}
