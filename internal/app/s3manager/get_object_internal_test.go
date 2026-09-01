package s3manager

import (
	"testing"

	"github.com/matryer/is"
)

func TestInlineContentType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it          string
		contentType string
		expected    string
	}{
		{
			it:          "keeps a content type the browser can render safely",
			contentType: "application/pdf",
			expected:    "application/pdf",
		},
		{
			it:          "keeps parameters of a renderable content type",
			contentType: "text/plain; charset=iso-8859-1",
			expected:    "text/plain; charset=iso-8859-1",
		},
		{
			it:          "matches renderable content types case-insensitively",
			contentType: "Application/JSON",
			expected:    "Application/JSON",
		},
		{
			it:          "keeps any image type except SVG",
			contentType: "image/png",
			expected:    "image/png",
		},
		{
			it:          "keeps any video type",
			contentType: "video/mp4",
			expected:    "video/mp4",
		},
		{
			it:          "falls back to plain text for HTML",
			contentType: "text/html",
			expected:    "text/plain; charset=utf-8",
		},
		{
			it:          "falls back to plain text for SVG",
			contentType: "image/svg+xml",
			expected:    "text/plain; charset=utf-8",
		},
		{
			it:          "falls back to plain text for XML",
			contentType: "application/xml",
			expected:    "text/plain; charset=utf-8",
		},
		{
			it:          "falls back to plain text for an unknown content type",
			contentType: "application/octet-stream",
			expected:    "text/plain; charset=utf-8",
		},
		{
			it:          "falls back to plain text for a missing content type",
			contentType: "",
			expected:    "text/plain; charset=utf-8",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			is.Equal(tc.expected, inlineContentType(tc.contentType)) // content type
		})
	}
}
