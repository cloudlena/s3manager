package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sync"
)

// templateFuncs are the helpers the page templates rely on.
var templateFuncs = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
	"mul": func(a, b int) int { return a * b },
	"min": func(a, b int) int { return min(a, b) },
	"iterate": func(start, end int) []int {
		result := make([]int, 0, max(end-start, 0))
		for i := start; i < end; i++ {
			result = append(result, i)
		}
		return result
	},
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

// newPageRenderer returns a renderer for the given page template. The template
// is parsed together with the layout on first use and reused afterwards.
func newPageRenderer(templates fs.FS, page string) pageRenderer {
	parse := sync.OnceValues(func() (*template.Template, error) {
		t, err := template.New("").Funcs(templateFuncs).ParseFS(templates, "layout.html.tmpl", page)
		if err != nil {
			return nil, fmt.Errorf("error parsing template files: %w", err)
		}
		return t, nil
	})

	return func(w http.ResponseWriter, data any) {
		t, err := parse()
		if err != nil {
			handleHTTPError(w, err)
			return
		}

		if err := t.ExecuteTemplate(w, "layout", data); err != nil {
			handleHTTPError(w, fmt.Errorf("error executing template: %w", err))
		}
	}
}
