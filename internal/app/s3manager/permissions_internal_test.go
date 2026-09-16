package s3manager

import (
	"context"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager/auth"
	"github.com/matryer/is"
)

// allRoles is every role a request can carry.
var allRoles = []auth.Role{auth.RoleNone, auth.RoleViewer, auth.RoleWriter}

// contextWithRole returns a request context carrying the given role.
func contextWithRole(role auth.Role) context.Context {
	return auth.WithIdentity(context.Background(), &auth.Identity{Role: role})
}

// TestEffectiveOptionsNeverGrants guards the rule the whole authorization model
// rests on: the configured options are a global cap, so no role may ever turn a
// capability on that the deployment turned off. Combining a capability with the
// role rather than assigning the role to it keeps this true, which is why every
// line of effectiveOptions is an AND.
func TestEffectiveOptionsNeverGrants(t *testing.T) {
	is := is.New(t)

	for _, role := range allRoles {
		is.Equal(effectiveOptions(contextWithRole(role), Options{}), Options{}) // a role must not enable anything
	}
}

func TestEffectiveOptionsDeleteNeedsWriter(t *testing.T) {
	tests := []struct {
		name        string
		role        auth.Role
		allowDelete bool
		expected    bool
	}{
		{name: "writer keeps delete", role: auth.RoleWriter, allowDelete: true, expected: true},
		{name: "viewer loses delete", role: auth.RoleViewer, allowDelete: true, expected: false},
		{name: "unauthenticated loses delete", role: auth.RoleNone, allowDelete: true, expected: false},
		{name: "writer cannot gain delete", role: auth.RoleWriter, allowDelete: false, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			is := is.New(t)

			opts := effectiveOptions(contextWithRole(test.role), Options{AllowDelete: test.allowDelete})

			is.Equal(opts.AllowDelete, test.expected)
		})
	}
}

// TestEffectiveOptionsKeepsPresentation makes sure the settings that only
// decide how something is shown stay out of the authorization path.
func TestEffectiveOptionsKeepsPresentation(t *testing.T) {
	is := is.New(t)

	opts := Options{
		RootURL:       "/s3",
		BucketName:    "bucket",
		ForceDownload: true,
		ListRecursive: true,
		ShowVersions:  true,
		ShowMetadata:  true,
		SSE:           SSEType{Type: "SSE-S3"},
	}

	is.Equal(effectiveOptions(contextWithRole(auth.RoleViewer), opts), opts) // presentation is unchanged by the role
}
