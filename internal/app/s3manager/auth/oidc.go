package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

// The endpoints the OIDC authenticator mounts. They sit outside the protected
// part of the app, because a user must be able to reach them to log in.
const (
	LoginPath    = "/auth/login"
	CallbackPath = "/auth/callback"
	LogoutPath   = "/auth/logout"
)

// OIDCConfig configures the OpenID Connect authenticator.
type OIDCConfig struct {
	// Issuer is the provider's issuer URL, from which its configuration is
	// discovered.
	Issuer string
	// ClientID and ClientSecret identify this app at the provider.
	ClientID     string
	ClientSecret string
	// RedirectURL is the absolute URL the provider sends the user back to. It
	// must be the app's public URL followed by CallbackPath.
	RedirectURL string
	// Scopes are requested in addition to openid.
	Scopes []string
	// RoleClaim is the ID token claim the role is derived from.
	RoleClaim string
	// ViewerGroups and WriterGroups are the claim values granting each role. A
	// user matching both gets the higher one.
	ViewerGroups []string
	WriterGroups []string
	// DefaultRole applies to a user matching no configured group. It is
	// RoleNone by default, which denies access.
	DefaultRole Role
	// SessionKey seals the session cookies. It must be 32 bytes.
	SessionKey []byte
	// SessionMaxAge is how long a session cookie stays valid, in seconds.
	SessionMaxAge int
	// RootURL is the path prefix the app is served under.
	RootURL string
	// CookieSecure marks the session cookies as HTTPS only.
	CookieSecure bool
}

// oidcAuthenticator logs users in through an OpenID Connect provider using the
// authorization code flow with PKCE, and keeps the result in a session cookie.
// It only ever authenticates access to the app; the S3 credentials are
// unrelated and stay server side.
type oidcAuthenticator struct {
	config   OIDCConfig
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
	sessions *sessionStore
	// endSessionURL is the provider's logout endpoint, if it advertises one.
	endSessionURL string
}

// NewOIDC discovers the provider's configuration and returns an authenticator
// for it. Discovery happens once at start-up, so a provider that is unreachable
// fails the app fast instead of at the first login.
func NewOIDC(ctx context.Context, config OIDCConfig) (Authenticator, error) {
	if len(config.SessionKey) != 32 {
		return nil, errors.New("session key must be 32 bytes")
	}

	provider, err := oidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, fmt.Errorf("error discovering OIDC provider %s: %w", config.Issuer, err)
	}

	// The end session endpoint is not part of go-oidc's provider struct, so it
	// is read from the raw discovery document.
	var discovery struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := provider.Claims(&discovery); err != nil {
		return nil, fmt.Errorf("error reading OIDC discovery document: %w", err)
	}

	return &oidcAuthenticator{
		config:   config,
		verifier: provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
		oauth: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  config.RedirectURL,
			Scopes:       append([]string{oidc.ScopeOpenID}, config.Scopes...),
		},
		sessions:      newSessionStore(config.SessionKey, config.RootURL, config.CookieSecure),
		endSessionURL: discovery.EndSessionEndpoint,
	}, nil
}

// Authenticate returns the identity of the session cookie. A request without a
// valid session is sent to the provider, or answered with an unauthorized
// status if it came from the app's own JavaScript rather than from navigation.
func (a *oidcAuthenticator) Authenticate(w http.ResponseWriter, r *http.Request) (*Identity, bool) {
	if identity := a.sessions.identity(r); identity != nil {
		return identity, true
	}

	loginURL := a.config.RootURL + LoginPath
	if isAPIRequest(r) {
		writeUnauthorized(w, loginURL)
		return nil, false
	}

	http.Redirect(w, r, loginURL+"?return_to="+url.QueryEscape(a.config.RootURL+r.URL.RequestURI()), http.StatusFound)

	return nil, false
}

// RegisterRoutes mounts the login, callback and logout endpoints.
func (a *oidcAuthenticator) RegisterRoutes(router *mux.Router) {
	router.HandleFunc(LoginPath, a.handleLogin).Methods(http.MethodGet)
	router.HandleFunc(CallbackPath, a.handleCallback).Methods(http.MethodGet)
	router.HandleFunc(LogoutPath, a.handleLogout).Methods(http.MethodGet, http.MethodPost)
}

// handleLogin starts the authorization code flow.
func (a *oidcAuthenticator) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomString()
	if err != nil {
		httpError(w, "error starting login", err)
		return
	}
	nonce, err := randomString()
	if err != nil {
		httpError(w, "error starting login", err)
		return
	}
	verifier := oauth2.GenerateVerifier()

	pending := login{
		State:    state,
		Nonce:    nonce,
		Verifier: verifier,
		ReturnTo: a.safeReturnTo(r.URL.Query().Get("return_to")),
	}
	if err := a.sessions.saveLogin(w, r, pending); err != nil {
		httpError(w, "error starting login", err)
		return
	}

	http.Redirect(w, r, a.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

// handleCallback completes the authorization code flow and opens the session.
func (a *oidcAuthenticator) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if providerError := query.Get("error"); providerError != "" {
		description := query.Get("error_description")
		http.Error(w, strings.TrimSpace("Login failed: "+providerError+" "+description), http.StatusUnauthorized)
		return
	}

	pending, err := a.sessions.takeLogin(w, r)
	if err != nil {
		httpError(w, "login expired, please try again", err)
		return
	}
	// A constant time comparison keeps the check from leaking the state value.
	if subtle.ConstantTimeCompare([]byte(pending.State), []byte(query.Get("state"))) != 1 {
		http.Error(w, "Login failed: state mismatch", http.StatusBadRequest)
		return
	}

	token, err := a.oauth.Exchange(r.Context(), query.Get("code"), oauth2.VerifierOption(pending.Verifier))
	if err != nil {
		httpError(w, "error exchanging authorization code", err)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "Login failed: provider returned no ID token", http.StatusBadGateway)
		return
	}
	idToken, err := a.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		httpError(w, "error verifying ID token", err)
		return
	}
	if subtle.ConstantTimeCompare([]byte(pending.Nonce), []byte(idToken.Nonce)) != 1 {
		http.Error(w, "Login failed: nonce mismatch", http.StatusBadRequest)
		return
	}

	identity, err := a.identityFromToken(idToken)
	if err != nil {
		httpError(w, "error reading ID token claims", err)
		return
	}
	if identity.Role == RoleNone {
		log.Printf("denied login of %s: no configured group in claim %q", identity.Subject, a.config.RoleClaim)
		http.Error(w, "Forbidden: your account has no access to this app", http.StatusForbidden)
		return
	}

	if err := a.sessions.saveIdentity(w, r, identity, a.config.SessionMaxAge); err != nil {
		httpError(w, "error opening session", err)
		return
	}

	http.Redirect(w, r, pending.ReturnTo, http.StatusFound)
}

// handleLogout closes the session and, if the provider supports it, sends the
// user on to end the session there as well.
func (a *oidcAuthenticator) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := a.sessions.clearIdentity(w, r); err != nil {
		httpError(w, "error closing session", err)
		return
	}

	target := a.config.RootURL + "/"
	if a.endSessionURL != "" {
		target = a.endSessionURL + "?client_id=" + url.QueryEscape(a.config.ClientID)
	}

	http.Redirect(w, r, target, http.StatusFound)
}

// identityFromToken reads the app's identity out of a verified ID token.
func (a *oidcAuthenticator) identityFromToken(idToken *oidc.IDToken) (*Identity, error) {
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("error decoding claims: %w", err)
	}

	identity := &Identity{
		Subject: idToken.Subject,
		Name:    firstString(claims, "name", "preferred_username", "email"),
		Email:   firstString(claims, "email"),
		Groups:  claimStrings(claims[a.config.RoleClaim]),
	}
	// The pages show the name, so there always has to be something to show.
	if identity.Name == "" {
		identity.Name = idToken.Subject
	}
	identity.Role = a.role(identity.Groups)

	return identity, nil
}

// role maps a user's group claim values to the role they get. The higher role
// wins, and a user matching nothing gets the configured default, which denies
// access unless the deployment says otherwise.
func (a *oidcAuthenticator) role(groups []string) Role {
	if slices.ContainsFunc(groups, func(g string) bool { return slices.Contains(a.config.WriterGroups, g) }) {
		return RoleWriter
	}
	if slices.ContainsFunc(groups, func(g string) bool { return slices.Contains(a.config.ViewerGroups, g) }) {
		return RoleViewer
	}

	return a.config.DefaultRole
}

// safeReturnTo accepts only paths inside this app, so that the login endpoint
// cannot be used to bounce a user to another site.
func (a *oidcAuthenticator) safeReturnTo(returnTo string) string {
	fallback := a.config.RootURL + "/"

	// A protocol relative path such as //evil.example or /\evil.example points
	// at another host even though it starts with a slash: some browsers
	// normalize a leading backslash to a second forward slash.
	if returnTo == "" || strings.ContainsRune(returnTo, '\\') ||
		!strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
		return fallback
	}
	parsed, err := url.Parse(returnTo)
	if err != nil || parsed.Host != "" || parsed.Scheme != "" {
		return fallback
	}
	// Sending the user straight back to a login endpoint would loop.
	if strings.HasPrefix(parsed.Path, a.config.RootURL+"/auth/") {
		return fallback
	}

	return returnTo
}

// randomString returns 32 bytes of randomness, URL safe.
func randomString() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("error reading random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// firstString returns the first of the given claims that holds a non-empty
// string.
func firstString(claims map[string]any, names ...string) string {
	for _, name := range names {
		if value, ok := claims[name].(string); ok && value != "" {
			return value
		}
	}

	return ""
}

// claimStrings normalises a group claim, which providers deliver either as a
// list or as a single space separated string.
func claimStrings(claim any) []string {
	switch value := claim.(type) {
	case []any:
		values := make([]string, 0, len(value))
		for _, entry := range value {
			if s, ok := entry.(string); ok && s != "" {
				values = append(values, s)
			}
		}
		return values
	case []string:
		return value
	case string:
		return strings.Fields(value)
	default:
		return nil
	}
}

// isAPIRequest reports whether a request came from the app's own JavaScript, in
// which case a redirect to the provider would be useless.
func isAPIRequest(r *http.Request) bool {
	return strings.Contains(r.URL.Path, "/api/") ||
		strings.Contains(r.Header.Get("Accept"), "application/json") ||
		r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

// writeUnauthorized tells the front end where to send the user to log in. The
// status code is already written when encoding fails, so such an error can only
// be logged.
func writeUnauthorized(w http.ResponseWriter, loginURL string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)

	body := struct {
		Error    string `json:"error"`
		LoginURL string `json:"loginUrl"`
	}{Error: "unauthenticated", LoginURL: loginURL}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

// httpError logs the cause and tells the user only what they can act on.
func httpError(w http.ResponseWriter, message string, err error) {
	log.Printf("auth: %s: %s", message, err)
	http.Error(w, "Login failed: "+message, http.StatusInternalServerError)
}
