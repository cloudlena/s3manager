package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matryer/is"
)

// TestSafeReturnTo covers the open redirect guard on the login endpoint: a
// crafted link must never be able to bounce a user to another site.
func TestSafeReturnTo(t *testing.T) {
	tests := []struct {
		name     string
		rootURL  string
		returnTo string
		expected string
	}{
		{name: "app path", returnTo: "/minio/buckets/photos", expected: "/minio/buckets/photos"},
		{name: "path with query", returnTo: "/minio/buckets?sortBy=size", expected: "/minio/buckets?sortBy=size"},
		{name: "empty falls back", returnTo: "", expected: "/"},
		{name: "absolute URL rejected", returnTo: "https://evil.example/steal", expected: "/"},
		{name: "protocol relative rejected", returnTo: "//evil.example/steal", expected: "/"},
		{name: "scheme relative rejected", returnTo: "javascript:alert(1)", expected: "/"},
		{name: "relative path rejected", returnTo: "buckets", expected: "/"},
		{name: "login path would loop", returnTo: "/auth/login", expected: "/"},
		{name: "callback path would loop", returnTo: "/auth/callback?code=x", expected: "/"},
		{name: "root URL is kept", rootURL: "/s3", returnTo: "/s3/minio/buckets", expected: "/s3/minio/buckets"},
		{name: "root URL fallback", rootURL: "/s3", returnTo: "https://evil.example", expected: "/s3/"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			is := is.New(t)

			authenticator := &oidcAuthenticator{config: OIDCConfig{RootURL: test.rootURL}}

			is.Equal(authenticator.safeReturnTo(test.returnTo), test.expected)
		})
	}
}

// TestRole checks how the group claim maps to a role, including that the
// higher role wins and that an unknown user is denied by default.
func TestRole(t *testing.T) {
	config := OIDCConfig{
		ViewerGroups: []string{"s3-readers", "everyone"},
		WriterGroups: []string{"s3-admins"},
	}

	tests := []struct {
		name     string
		groups   []string
		expected Role
	}{
		{name: "viewer group", groups: []string{"s3-readers"}, expected: RoleViewer},
		{name: "writer group", groups: []string{"s3-admins"}, expected: RoleWriter},
		{name: "both groups gives the higher role", groups: []string{"s3-readers", "s3-admins"}, expected: RoleWriter},
		{name: "unknown group is denied", groups: []string{"marketing"}, expected: RoleNone},
		{name: "no group is denied", groups: nil, expected: RoleNone},
		{name: "unrelated groups are ignored", groups: []string{"marketing", "everyone"}, expected: RoleViewer},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			is := is.New(t)

			authenticator := &oidcAuthenticator{config: config}

			is.Equal(authenticator.role(test.groups), test.expected)
		})
	}
}

// TestRoleDefault covers a deployment that deliberately lets everybody the
// provider authenticates in with a base role.
func TestRoleDefault(t *testing.T) {
	is := is.New(t)

	authenticator := &oidcAuthenticator{config: OIDCConfig{
		WriterGroups: []string{"s3-admins"},
		DefaultRole:  RoleViewer,
	}}

	is.Equal(authenticator.role([]string{"marketing"}), RoleViewer)
	is.Equal(authenticator.role([]string{"s3-admins"}), RoleWriter)
}

// TestClaimStrings covers the shapes providers deliver a group claim in.
func TestClaimStrings(t *testing.T) {
	tests := []struct {
		name     string
		claim    any
		expected []string
	}{
		{name: "list", claim: []any{"a", "b"}, expected: []string{"a", "b"}},
		{name: "list of strings", claim: []string{"a", "b"}, expected: []string{"a", "b"}},
		{name: "space separated", claim: "a b", expected: []string{"a", "b"}},
		{name: "single value", claim: "a", expected: []string{"a"}},
		{name: "list with non strings", claim: []any{"a", 1, ""}, expected: []string{"a"}},
		{name: "missing", claim: nil, expected: nil},
		{name: "wrong type", claim: 42, expected: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			is := is.New(t)

			is.Equal(claimStrings(test.claim), test.expected)
		})
	}
}

func TestFirstString(t *testing.T) {
	is := is.New(t)

	claims := map[string]any{"preferred_username": "jdoe", "email": "j@example.com", "name": ""}

	is.Equal(firstString(claims, "name", "preferred_username", "email"), "jdoe") // an empty claim is skipped
	is.Equal(firstString(claims, "missing"), "")
}

// TestIsAPIRequest checks which requests get an unauthorized status instead of
// a redirect, because a redirect to the provider is useless to a fetch call.
func TestIsAPIRequest(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		header   map[string]string
		expected bool
	}{
		{name: "page", path: "/minio/buckets", expected: false},
		{name: "api path", path: "/minio/api/buckets", expected: true},
		{name: "json accept header", path: "/minio/buckets", header: map[string]string{"Accept": "application/json"}, expected: true},
		{name: "XHR header", path: "/minio/buckets", header: map[string]string{"X-Requested-With": "XMLHttpRequest"}, expected: true},
		{name: "browser accept header", path: "/minio/buckets", header: map[string]string{"Accept": "text/html"}, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			is := is.New(t)

			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			for name, value := range test.header {
				req.Header.Set(name, value)
			}

			is.Equal(isAPIRequest(req), test.expected)
		})
	}
}

// TestAuthenticateWithoutSession covers what an unauthenticated request gets:
// a page request is sent to the login endpoint, a request from the app's own
// JavaScript gets an unauthorized status it can act on.
func TestAuthenticateWithoutSession(t *testing.T) {
	is := is.New(t)

	authenticator := &oidcAuthenticator{
		config:   OIDCConfig{RootURL: "/s3"},
		sessions: newSessionStore(deriveSessionKey("secret"), "/s3", true),
	}

	t.Run("page is redirected", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/minio/buckets", nil)

		identity, ok := authenticator.Authenticate(resp, req)

		is.True(!ok) // the request must not be handled
		is.Equal(identity, (*Identity)(nil))
		is.Equal(resp.Code, http.StatusFound)
		is.Equal(resp.Header().Get("Location"), "/s3/auth/login?return_to=%2Fs3%2Fminio%2Fbuckets")
	})

	t.Run("API call is refused", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/minio/api/buckets/photos", nil)

		_, ok := authenticator.Authenticate(resp, req)

		is.True(!ok) // the request must not be handled
		is.Equal(resp.Code, http.StatusUnauthorized)
		is.Equal(strings.TrimSpace(resp.Body.String()), `{"error":"unauthenticated","loginUrl":"/s3/auth/login"}`)
	})
}
