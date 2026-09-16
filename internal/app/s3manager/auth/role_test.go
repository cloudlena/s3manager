package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager/auth"
	"github.com/matryer/is"
)

func TestParseRole(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  auth.Role
		shouldErr bool
	}{
		{name: "viewer", value: "viewer", expected: auth.RoleViewer},
		{name: "writer", value: "writer", expected: auth.RoleWriter},
		{name: "none", value: "none", expected: auth.RoleNone},
		{name: "case insensitive", value: "Writer", expected: auth.RoleWriter},
		{name: "unknown", value: "admin", shouldErr: true},
		{name: "empty", value: "", shouldErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			is := is.New(t)

			role, err := auth.ParseRole(test.value)

			if test.shouldErr {
				is.True(err != nil) // should return an error
				return
			}
			is.NoErr(err)
			is.Equal(role, test.expected)
		})
	}
}

// TestRoleOrder pins the ordering the permission checks depend on.
func TestRoleOrder(t *testing.T) {
	is := is.New(t)

	is.True(auth.RoleNone < auth.RoleViewer)   // an unauthenticated request is the weakest
	is.True(auth.RoleViewer < auth.RoleWriter) // a writer may do everything a viewer may
}

// TestRoleFromEmptyContext makes sure a request that never passed an
// authenticator is denied rather than granted.
func TestRoleFromEmptyContext(t *testing.T) {
	is := is.New(t)

	is.Equal(auth.RoleFrom(context.Background()), auth.RoleNone)
	is.Equal(auth.IdentityFrom(context.Background()), (*auth.Identity)(nil))
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name     string
		role     auth.Role
		minimum  auth.Role
		expected int
	}{
		{name: "writer passes writer guard", role: auth.RoleWriter, minimum: auth.RoleWriter, expected: http.StatusOK},
		{name: "viewer fails writer guard", role: auth.RoleViewer, minimum: auth.RoleWriter, expected: http.StatusForbidden},
		{name: "viewer passes viewer guard", role: auth.RoleViewer, minimum: auth.RoleViewer, expected: http.StatusOK},
		{name: "unauthenticated fails viewer guard", role: auth.RoleNone, minimum: auth.RoleViewer, expected: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			is := is.New(t)

			handler := auth.RequireRole(test.minimum)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{Role: test.role}))
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			is.Equal(resp.Code, test.expected)
		})
	}
}

// TestRequireRoleWithoutIdentity makes sure the guard denies a request that
// reached it without going through an authenticator.
func TestRequireRoleWithoutIdentity(t *testing.T) {
	is := is.New(t)

	handler := auth.RequireRole(auth.RoleViewer)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/", nil))

	is.Equal(resp.Code, http.StatusForbidden)
}
