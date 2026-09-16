package auth

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Authenticator establishes the identity behind a request. Implementations keep
// their own state and their own user list; the app only sees the resulting
// identity. Adding another login mechanism means adding an implementation, not
// changing the handlers.
type Authenticator interface {
	// Authenticate returns the identity of the request. When it returns false
	// it has already written a response, such as a redirect to a login page or
	// an authentication challenge, and the request must not be handled further.
	Authenticate(w http.ResponseWriter, r *http.Request) (*Identity, bool)

	// RegisterRoutes mounts the endpoints the authenticator needs to complete a
	// login, such as a redirect target or a logout route. These routes are
	// always reachable without being authenticated.
	RegisterRoutes(router *mux.Router)
}

// Middleware returns middleware that authenticates every request and puts the
// resulting identity into its context.
func Middleware(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := authenticator.Authenticate(w, r)
			if !ok {
				return
			}

			next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), identity)))
		})
	}
}

// anonymous authenticates nobody and grants everybody the same role. It backs
// the unauthenticated deployment, where the configured feature flags alone
// decide what the app offers.
type anonymous struct {
	identity *Identity
}

// NewAnonymous returns an authenticator that lets every request through with
// the given role. It is what AUTH_PROVIDER=none installs, so that a request
// always carries an explicit role and no handler has to fall back to a default.
func NewAnonymous(role Role) Authenticator {
	return &anonymous{identity: &Identity{Subject: "anonymous", Role: role}}
}

// Authenticate grants every request the same identity.
func (a *anonymous) Authenticate(_ http.ResponseWriter, _ *http.Request) (*Identity, bool) {
	return a.identity, true
}

// RegisterRoutes mounts nothing, as there is nothing to log in to.
func (a *anonymous) RegisterRoutes(_ *mux.Router) {}
