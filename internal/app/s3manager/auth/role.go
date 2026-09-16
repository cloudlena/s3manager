// Package auth authenticates and authorizes access to the web app. It is
// independent of the S3 credentials the app uses to talk to storage: those stay
// server side and are never derived from the logged in user.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Role is what an authenticated user is allowed to do in the app. Roles are
// ordered, so a check reads as `role >= RoleWriter`.
type Role int

const (
	// RoleNone has no access at all. It is the zero value so that a request
	// which never passed an authenticator is denied rather than allowed.
	RoleNone Role = iota
	// RoleViewer may list buckets, view, download and share objects.
	RoleViewer
	// RoleWriter may additionally create and delete buckets and objects.
	RoleWriter
)

// roleNames maps a role to its configuration value.
var roleNames = map[Role]string{
	RoleNone:   "none",
	RoleViewer: "viewer",
	RoleWriter: "writer",
}

// String returns the configuration value of the role.
func (r Role) String() string {
	name, ok := roleNames[r]
	if !ok {
		return "unknown"
	}

	return name
}

// ParseRole reads a role from its configuration value.
func ParseRole(s string) (Role, error) {
	for role, name := range roleNames {
		if strings.EqualFold(s, name) {
			return role, nil
		}
	}

	return RoleNone, fmt.Errorf("invalid role: %s", s)
}

// contextKey is the private type of this package's context keys.
type contextKey struct{}

// identityKey addresses the identity of the current request.
var identityKey = contextKey{}

// Identity is the authenticated principal behind a request.
type Identity struct {
	// Subject uniquely identifies the user at the identity provider.
	Subject string
	// Name is the user's display name, if the provider supplied one.
	Name string
	// Email is the user's email address, if the provider supplied one.
	Email string
	// Groups are the raw claim values the role was derived from.
	Groups []string
	// Role is what the user may do in the app.
	Role Role
}

// WithIdentity returns a copy of the context carrying the given identity.
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

// IdentityFrom returns the identity of the request's context, or nil if the
// request never passed an authenticator.
func IdentityFrom(ctx context.Context) *Identity {
	identity, _ := ctx.Value(identityKey).(*Identity)

	return identity
}

// RoleFrom returns the role of the request's context. A request without an
// identity has RoleNone, so a missing authenticator denies rather than grants.
func RoleFrom(ctx context.Context) Role {
	identity := IdentityFrom(ctx)
	if identity == nil {
		return RoleNone
	}

	return identity.Role
}

// RequireRole returns middleware that rejects requests whose role is below the
// given minimum. It narrows access within what the deployment already allows;
// it can never widen it, because a route the configuration disabled is not
// registered in the first place.
func RequireRole(minimum Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if RoleFrom(r.Context()) < minimum {
				http.Error(w, fmt.Sprintf("Forbidden: %s access required", minimum), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
