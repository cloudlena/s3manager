package s3manager_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/s3manager"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/matryer/is"
)

func TestHandleBucketsView(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:                   "renders the buckets page",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "S3 Manager",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			server := s3manager.New(nil, true, "", "")

			engine := html.New("../../views", ".html.gotmpl")
			app := fiber.New(fiber.Config{
				Views: engine,
			})
			app.Get("/buckets", server.HandleBucketsView)

			req, err := http.NewRequest(fiber.MethodGet, "/buckets", nil)
			is.NoErr(err)

			resp, err := app.Test(req)
			is.NoErr(err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(resp.StatusCode, tc.expectedStatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}
