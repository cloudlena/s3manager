package s3manager

import (
	"context"

	"github.com/cloudlena/s3manager/internal/app/s3manager/auth"
)

// effectiveOptions narrows the configured options to what the role of the
// request allows.
//
// The configured options are a global cap: a role can only ever take something
// away, never grant something the deployment disabled. Every capability here is
// therefore combined with the role, never assigned from it.
//
// This result drives what the pages offer. It is not the security boundary:
// routes that need a role are guarded with auth.RequireRole, and routes the
// configuration disables are not registered at all.
func effectiveOptions(ctx context.Context, opts Options) Options {
	isWriter := auth.RoleFrom(ctx) >= auth.RoleWriter

	opts.AllowDelete = opts.AllowDelete && isWriter

	return opts
}

// userName is the name the pages label the session with. It is empty when the
// deployment authenticates nobody, which is what hides the log out action.
func userName(ctx context.Context) string {
	identity := auth.IdentityFrom(ctx)
	if identity == nil {
		return ""
	}

	return identity.Name
}
