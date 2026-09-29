package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"
)

// timeLayout is how the app shows the time an object was last modified.
const timeLayout = "2006-01-02 15:04:05 MST"

// formatTime formats a time for display in the app's time zone, which TZ sets.
func formatTime(t time.Time) string {
	return t.Local().Format(timeLayout)
}

// templateFuncs are the helpers the page templates rely on.
var templateFuncs = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
	// escapeKey escapes an object key for use in a link. html/template alone
	// leaves "#", "?" and anything resembling a percent-escape untouched, so
	// such keys would otherwise address a different object.
	"escapeKey":  escapeObjectKey,
	"formatTime": formatTime,
	// sortIndicator names the icon a sortable table header shows, or nothing
	// if the table is not sorted by that column.
	"sortIndicator": func(field, sortBy, sortOrder string) string {
		switch {
		case field != sortBy:
			return ""
		case sortOrder == "desc":
			return "arrow_downward"
		default:
			return "arrow_upward"
		}
	},
}

// pageRenderer renders one page template with the data of a single request.
type pageRenderer func(w http.ResponseWriter, data any)

// newPageRenderer returns a renderer for the given page template, parsed
// together with the layout. The templates are embedded into the binary, so one
// that fails to parse is a programming error, which panics right at start-up.
func newPageRenderer(templates fs.FS, page string) pageRenderer {
	t := template.Must(template.New("").Funcs(templateFuncs).ParseFS(templates, "layout.html.tmpl", page))

	return func(w http.ResponseWriter, data any) {
		if err := t.ExecuteTemplate(w, "layout", data); err != nil {
			handleHTTPError(w, fmt.Errorf("error executing template: %w", err))
		}
	}
}
