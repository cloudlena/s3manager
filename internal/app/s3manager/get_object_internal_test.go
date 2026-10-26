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
		inline      bool
	}{
		{
			it:          "keeps a content type the browser can render safely",
			contentType: "application/pdf",
			expected:    "application/pdf",
			inline:      true,
		},
		{
			it:          "keeps parameters of a renderable content type",
			contentType: "text/plain; charset=iso-8859-1",
			expected:    "text/plain; charset=iso-8859-1",
			inline:      true,
		},
		{
			it:          "matches renderable content types case-insensitively",
			contentType: "Application/JSON",
			expected:    "Application/JSON",
			inline:      true,
		},
		{
			it:          "keeps any image type except SVG",
			contentType: "image/png",
			expected:    "image/png",
			inline:      true,
		},
		{
			it:          "keeps any video type",
			contentType: "video/mp4",
			expected:    "video/mp4",
			inline:      true,
		},
		{
			it:          "falls back to plain text for HTML",
			contentType: "text/html",
			expected:    "text/plain; charset=utf-8",
			inline:      true,
		},
		{
			it:          "falls back to plain text for SVG",
			contentType: "image/svg+xml",
			expected:    "text/plain; charset=utf-8",
			inline:      true,
		},
		{
			it:          "falls back to plain text for XML",
			contentType: "application/xml",
			expected:    "text/plain; charset=utf-8",
			inline:      true,
		},
		{
			it:          "falls back to plain text for any other text type",
			contentType: "text/x-go; charset=utf-8",
			expected:    "text/plain; charset=utf-8",
			inline:      true,
		},
		{
			it:          "falls back to plain text for structured syntax suffixes",
			contentType: "application/ld+json",
			expected:    "text/plain; charset=utf-8",
			inline:      true,
		},
		{
			it:          "can't display a spreadsheet",
			contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		},
		{
			it:          "can't display an unknown binary content type",
			contentType: "application/octet-stream",
		},
		{
			it:          "can't display an object without a content type",
			contentType: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			contentType, inline := inlineContentType(tc.contentType)
			is.Equal(tc.expected, contentType) // content type
			is.Equal(tc.inline, inline)        // inline
		})
	}
}
