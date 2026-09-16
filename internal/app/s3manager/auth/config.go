package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
)

// Provider names the login mechanism a deployment uses.
const (
	// ProviderNone leaves the app open, which is the historical behaviour.
	ProviderNone = "none"
	// ProviderOIDC logs users in through an OpenID Connect provider.
	ProviderOIDC = "oidc"
)

// Config is the full authentication configuration of a deployment.
type Config struct {
	// Provider selects the login mechanism.
	Provider string
	// AnonymousRole is what an unauthenticated user may do when Provider is
	// none. It is RoleWriter by default, so that the feature flags alone decide
	// what an open deployment offers, exactly as before authentication existed.
	AnonymousRole Role
	// SessionSecret seals the session cookies. Any length is accepted; the key
	// is derived from it. It must be the same across all replicas of the app.
	SessionSecret string
	// OIDC configures the OpenID Connect provider.
	OIDC OIDCConfig
}

// New returns the authenticator the configuration asks for.
func New(ctx context.Context, config Config) (Authenticator, error) {
	switch strings.ToLower(config.Provider) {
	case "", ProviderNone:
		return NewAnonymous(config.AnonymousRole), nil
	case ProviderOIDC:
		return newOIDCFromConfig(ctx, config)
	default:
		return nil, fmt.Errorf("invalid AUTH_PROVIDER: %s", config.Provider)
	}
}

// newOIDCFromConfig validates the OIDC settings and builds the authenticator.
func newOIDCFromConfig(ctx context.Context, config Config) (Authenticator, error) {
	oidcConfig := config.OIDC

	switch {
	case oidcConfig.Issuer == "":
		return nil, errors.New("OIDC_ISSUER is required when AUTH_PROVIDER is oidc")
	case oidcConfig.ClientID == "":
		return nil, errors.New("OIDC_CLIENT_ID is required when AUTH_PROVIDER is oidc")
	case oidcConfig.RedirectURL == "":
		return nil, errors.New("OIDC_REDIRECT_URL is required when AUTH_PROVIDER is oidc")
	case config.SessionSecret == "":
		// A generated key would silently log everybody out on every restart and
		// break any deployment running more than one replica.
		return nil, errors.New("SESSION_SECRET is required when authentication is enabled")
	}
	if oidcConfig.DefaultRole > RoleNone && len(oidcConfig.ViewerGroups) == 0 && len(oidcConfig.WriterGroups) == 0 {
		return nil, errors.New("OIDC_DEFAULT_ROLE grants access to everyone the provider authenticates; configure OIDC_VIEWER_GROUPS or OIDC_WRITER_GROUPS, or set it to none")
	}

	oidcConfig.SessionKey = deriveSessionKey(config.SessionSecret)

	return NewOIDC(ctx, oidcConfig)
}

// deriveSessionKey turns a secret of any length into the 32 byte key the
// cookie store needs.
func deriveSessionKey(secret string) []byte {
	key := sha256.Sum256([]byte(secret))

	return key[:]
}
