package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager/auth"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
)

// stubAuthenticator refuses every request, standing in for an authenticator
// that would send the user off to log in.
type stubAuthenticator struct {
	called bool
}

func (s *stubAuthenticator) Authenticate(w http.ResponseWriter, _ *http.Request) (*auth.Identity, bool) {
	s.called = true
	w.WriteHeader(http.StatusFound)

	return nil, false
}

func (s *stubAuthenticator) RegisterRoutes(_ *mux.Router) {}

// TestMiddlewarePutsIdentityInContext checks that a handler behind the
// middleware can read who the user is.
func TestMiddlewarePutsIdentityInContext(t *testing.T) {
	is := is.New(t)

	var seen *auth.Identity
	handler := auth.Middleware(auth.NewAnonymous(auth.RoleViewer))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = auth.IdentityFrom(r.Context())
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	is.True(seen != nil) // the handler should see an identity
	is.Equal(seen.Role, auth.RoleViewer)
}

// TestMiddlewareStopsRejectedRequest makes sure a refused request never reaches
// the handler behind the middleware.
func TestMiddlewareStopsRejectedRequest(t *testing.T) {
	is := is.New(t)

	authenticator := &stubAuthenticator{}
	reached := false
	handler := auth.Middleware(authenticator)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reached = true
	}))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/", nil))

	is.True(authenticator.called) // the authenticator should have been asked
	is.True(!reached)             // the handler must not run
	is.Equal(resp.Code, http.StatusFound)
}

// TestAnonymousGrantsConfiguredRole covers the unauthenticated deployment,
// where every request carries an explicit role so that no handler has to fall
// back to a default.
func TestAnonymousGrantsConfiguredRole(t *testing.T) {
	is := is.New(t)

	authenticator, err := auth.New(context.Background(), auth.Config{
		Provider:      auth.ProviderNone,
		AnonymousRole: auth.RoleWriter,
	})
	is.NoErr(err)

	identity, ok := authenticator.Authenticate(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	is.True(ok) // an open deployment lets every request through
	is.Equal(identity.Role, auth.RoleWriter)
}

func TestNewRejectsUnknownProvider(t *testing.T) {
	is := is.New(t)

	_, err := auth.New(context.Background(), auth.Config{Provider: "ldap"})

	is.True(err != nil) // an unknown provider should fail at start-up
}

// TestNewOIDCRequiresConfiguration makes sure a half configured deployment
// fails loudly at start-up instead of at the first login.
func TestNewOIDCRequiresConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config auth.Config
	}{
		{
			name:   "no issuer",
			config: auth.Config{Provider: auth.ProviderOIDC, SessionSecret: "secret"},
		},
		{
			name: "no client ID",
			config: auth.Config{
				Provider:      auth.ProviderOIDC,
				SessionSecret: "secret",
				OIDC:          auth.OIDCConfig{Issuer: "https://example.com"},
			},
		},
		{
			name: "no redirect URL",
			config: auth.Config{
				Provider:      auth.ProviderOIDC,
				SessionSecret: "secret",
				OIDC:          auth.OIDCConfig{Issuer: "https://example.com", ClientID: "s3manager"},
			},
		},
		{
			name: "no session secret",
			config: auth.Config{
				Provider: auth.ProviderOIDC,
				OIDC: auth.OIDCConfig{
					Issuer:      "https://example.com",
					ClientID:    "s3manager",
					RedirectURL: "https://s3.example.com/auth/callback",
				},
			},
		},
		{
			name: "default role without any group",
			config: auth.Config{
				Provider:      auth.ProviderOIDC,
				SessionSecret: "secret",
				OIDC: auth.OIDCConfig{
					Issuer:      "https://example.com",
					ClientID:    "s3manager",
					RedirectURL: "https://s3.example.com/auth/callback",
					DefaultRole: auth.RoleWriter,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			is := is.New(t)

			_, err := auth.New(context.Background(), test.config)

			is.True(err != nil) // an incomplete configuration should fail
		})
	}
}
