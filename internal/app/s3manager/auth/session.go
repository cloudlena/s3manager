package auth

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
)

// Session cookie names. The login cookie only lives for the duration of a
// single trip to the identity provider; the user cookie carries the session.
const (
	userSessionName  = "s3manager_session"
	loginSessionName = "s3manager_login"
	// loginMaxAge bounds how long a login may take before its state expires.
	loginMaxAge = 10 * 60
)

// Keys of the values stored in the session cookies.
const (
	keySubject  = "subject"
	keyName     = "name"
	keyEmail    = "email"
	keyGroups   = "groups"
	keyRole     = "role"
	keyState    = "state"
	keyNonce    = "nonce"
	keyVerifier = "verifier"
	keyReturnTo = "returnTo"
)

// The session values are encoded with encoding/gob, which needs to know the
// concrete types it will find behind an interface.
func init() {
	gob.Register([]string(nil))
}

// errNoPendingLogin is returned when a callback arrives without the app having
// started a login for that browser.
var errNoPendingLogin = errors.New("no pending login")

// login is the state the app keeps while the user is at the identity provider.
type login struct {
	// State is the opaque value the provider echoes back, tying the callback
	// to this browser and defeating cross site request forgery.
	State string
	// Nonce is echoed back inside the ID token, binding the token to this login.
	Nonce string
	// Verifier is the PKCE code verifier whose challenge was sent along.
	Verifier string
	// ReturnTo is the path the user asked for before being sent to log in.
	ReturnTo string
}

// sessionStore reads and writes the app's session cookies. Sessions are held
// entirely in the cookie, so the app stays stateless and needs no database.
type sessionStore struct {
	store *sessions.CookieStore
	// path is the cookie path, which follows the app's root URL so that a
	// reverse proxied deployment does not leak its cookie to sibling apps.
	path string
	// secure marks the cookies as HTTPS only.
	secure bool
}

// newSessionStore returns a store that seals its cookies with the given key.
// The key must be 32 bytes, giving AES-256 encryption on top of the signature.
func newSessionStore(key []byte, path string, secure bool) *sessionStore {
	if path == "" {
		path = "/"
	}

	return &sessionStore{
		store:  sessions.NewCookieStore(key, key),
		path:   path,
		secure: secure,
	}
}

// identity returns the identity stored in the request's session cookie, or nil
// if there is none or the cookie could not be decrypted.
func (s *sessionStore) identity(r *http.Request) *Identity {
	session, err := s.store.Get(r, userSessionName)
	if err != nil || session.IsNew {
		return nil
	}

	subject, ok := session.Values[keySubject].(string)
	if !ok || subject == "" {
		return nil
	}
	role, ok := session.Values[keyRole].(int)
	if !ok {
		return nil
	}

	identity := &Identity{Subject: subject, Role: Role(role)}
	identity.Name, _ = session.Values[keyName].(string)
	identity.Email, _ = session.Values[keyEmail].(string)
	identity.Groups, _ = session.Values[keyGroups].([]string)

	return identity
}

// saveIdentity writes the identity to a fresh session cookie.
func (s *sessionStore) saveIdentity(w http.ResponseWriter, r *http.Request, identity *Identity, maxAge int) error {
	session, _ := s.store.Get(r, userSessionName)
	session.Options = s.cookieOptions(maxAge)
	session.Values = map[any]any{
		keySubject: identity.Subject,
		keyName:    identity.Name,
		keyEmail:   identity.Email,
		keyGroups:  identity.Groups,
		keyRole:    int(identity.Role),
	}

	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("error saving session: %w", err)
	}

	return nil
}

// clearIdentity expires the session cookie.
func (s *sessionStore) clearIdentity(w http.ResponseWriter, r *http.Request) error {
	session, _ := s.store.Get(r, userSessionName)
	session.Options = s.cookieOptions(-1)
	session.Values = map[any]any{}

	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("error clearing session: %w", err)
	}

	return nil
}

// saveLogin remembers the state of a login that is about to start.
func (s *sessionStore) saveLogin(w http.ResponseWriter, r *http.Request, l login) error {
	session, _ := s.store.Get(r, loginSessionName)
	session.Options = s.cookieOptions(loginMaxAge)
	session.Values = map[any]any{
		keyState:    l.State,
		keyNonce:    l.Nonce,
		keyVerifier: l.Verifier,
		keyReturnTo: l.ReturnTo,
	}

	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("error saving login state: %w", err)
	}

	return nil
}

// takeLogin returns the pending login state and expires its cookie, so that a
// callback can only ever be completed once.
func (s *sessionStore) takeLogin(w http.ResponseWriter, r *http.Request) (login, error) {
	session, err := s.store.Get(r, loginSessionName)
	if err != nil {
		return login{}, fmt.Errorf("error reading login state: %w", err)
	}
	if session.IsNew {
		return login{}, errNoPendingLogin
	}

	var l login
	l.State, _ = session.Values[keyState].(string)
	l.Nonce, _ = session.Values[keyNonce].(string)
	l.Verifier, _ = session.Values[keyVerifier].(string)
	l.ReturnTo, _ = session.Values[keyReturnTo].(string)

	session.Options = s.cookieOptions(-1)
	session.Values = map[any]any{}
	if err := session.Save(r, w); err != nil {
		return login{}, fmt.Errorf("error clearing login state: %w", err)
	}

	if l.State == "" {
		return login{}, errNoPendingLogin
	}

	return l, nil
}

// cookieOptions returns the attributes every session cookie is written with.
func (s *sessionStore) cookieOptions(maxAge int) *sessions.Options {
	return &sessions.Options{
		Path:   s.path,
		MaxAge: maxAge,
		// The cookie is never read from JavaScript.
		HttpOnly: true,
		// Lax still sends the cookie on the top level redirect back from the
		// identity provider, which Strict would break.
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	}
}
