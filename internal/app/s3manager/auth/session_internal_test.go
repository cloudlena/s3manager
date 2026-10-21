package auth

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/matryer/is"
)

// replay turns the cookies a response set into a new request, standing in for
// the browser sending them back.
func replay(t *testing.T, resp *httptest.ResponseRecorder) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Result().Cookies() {
		req.AddCookie(cookie)
	}

	return req
}

func TestSessionIdentityRoundTrip(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true, 3600)
	identity := &Identity{
		Subject: "abc-123",
		Name:    "Jane Doe",
		Email:   "jane@example.com",
		Role:    RoleWriter,
	}

	resp := httptest.NewRecorder()
	is.NoErr(store.saveIdentity(resp, httptest.NewRequest(http.MethodGet, "/", nil), identity))

	is.Equal(store.identity(replay(t, resp)), identity)
}

func TestSessionWithoutCookie(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true, 3600)

	is.Equal(store.identity(httptest.NewRequest(http.MethodGet, "/", nil)), (*Identity)(nil))
}

// TestSessionRejectsForeignKey makes sure a cookie sealed with another key is
// not accepted, which is what stops a user from forging their own role.
func TestSessionRejectsForeignKey(t *testing.T) {
	is := is.New(t)

	resp := httptest.NewRecorder()
	writer := newSessionStore(deriveSessionKey("one secret"), "/", true, 3600)
	is.NoErr(writer.saveIdentity(resp, httptest.NewRequest(http.MethodGet, "/", nil), &Identity{Subject: "abc", Role: RoleWriter}))

	reader := newSessionStore(deriveSessionKey("another secret"), "/", true, 3600)

	is.Equal(reader.identity(replay(t, resp)), (*Identity)(nil))
}

func TestSessionCookieAttributes(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/s3", true, 3600)
	resp := httptest.NewRecorder()
	is.NoErr(store.saveIdentity(resp, httptest.NewRequest(http.MethodGet, "/", nil), &Identity{Subject: "abc", Role: RoleViewer}))

	cookie := resp.Result().Cookies()[0]

	is.Equal(cookie.Name, userSessionName)
	is.Equal(cookie.Path, "/s3")
	is.True(cookie.HttpOnly)                        // the cookie is never read from JavaScript
	is.True(cookie.Secure)                          // the cookie is only sent over HTTPS
	is.Equal(cookie.SameSite, http.SameSiteLaxMode) // strict would break the redirect back from the provider
}

// TestSessionExpires makes sure the server refuses a session older than the
// configured maximum age, rather than trusting the browser to drop the cookie.
func TestSessionExpires(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true, 1)
	resp := httptest.NewRecorder()
	is.NoErr(store.saveIdentity(resp, httptest.NewRequest(http.MethodGet, "/", nil), &Identity{Subject: "abc", Role: RoleWriter}))

	is.True(store.identity(replay(t, resp)) != nil) // a fresh session should be accepted

	// The timestamp inside the cookie has a resolution of one second, so
	// waiting just over two guarantees it is older than the one second limit.
	time.Sleep(2100 * time.Millisecond)

	// A new request is needed, as the store caches what it decoded per request.
	is.Equal(store.identity(replay(t, resp)), (*Identity)(nil)) // an expired session should be refused
}

// TestSessionMaxAgeAppliesToCookie makes sure the configured age reaches the
// cookie, and that the short lived login state keeps its own, shorter age.
func TestSessionMaxAgeAppliesToCookie(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true, 1234)

	session := httptest.NewRecorder()
	is.NoErr(store.saveIdentity(session, httptest.NewRequest(http.MethodGet, "/", nil), &Identity{Subject: "abc", Role: RoleViewer}))
	is.Equal(session.Result().Cookies()[0].MaxAge, 1234)

	pending := httptest.NewRecorder()
	is.NoErr(store.saveLogin(pending, httptest.NewRequest(http.MethodGet, "/", nil), login{State: "state-value"}))
	is.Equal(pending.Result().Cookies()[0].MaxAge, loginMaxAge)
	is.Equal(store.logins.Options.MaxAge, loginMaxAge)
	is.Equal(store.users.Options.MaxAge, 1234)
}

// TestSessionIgnoresGroups makes sure a user in a great many groups can still
// log in. Storing the groups would push the cookie past the 4096 bytes
// securecookie allows and fail the login.
func TestSessionIgnoresGroups(t *testing.T) {
	is := is.New(t)

	var groups []string
	for i := 0; i < 500; i++ {
		groups = append(groups, "group-"+strconv.Itoa(i))
	}
	authenticator := &oidcAuthenticator{config: OIDCConfig{WriterGroups: []string{"group-499"}}}
	role := authenticator.role(groups)
	is.Equal(role, RoleWriter) // the role is still derived from the groups

	store := newSessionStore(deriveSessionKey("secret"), "/", true, 3600)
	resp := httptest.NewRecorder()
	is.NoErr(store.saveIdentity(resp, httptest.NewRequest(http.MethodGet, "/", nil), &Identity{Subject: "abc", Role: role}))

	is.Equal(store.identity(replay(t, resp)).Role, RoleWriter)
}

func TestClearIdentity(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true, 3600)
	saved := httptest.NewRecorder()
	is.NoErr(store.saveIdentity(saved, httptest.NewRequest(http.MethodGet, "/", nil), &Identity{Subject: "abc", Role: RoleWriter}))

	cleared := httptest.NewRecorder()
	is.NoErr(store.clearIdentity(cleared, replay(t, saved)))

	is.Equal(store.identity(replay(t, cleared)), (*Identity)(nil))
}

func TestLoginRoundTrip(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true, 3600)
	pending := login{State: "state-value", Nonce: "nonce-value", Verifier: "verifier-value", ReturnTo: "/minio/buckets"}

	saved := httptest.NewRecorder()
	is.NoErr(store.saveLogin(saved, httptest.NewRequest(http.MethodGet, "/", nil), pending))

	taken, err := store.takeLogin(httptest.NewRecorder(), replay(t, saved))

	is.NoErr(err)
	is.Equal(taken, pending)
}

// TestTakeLoginWithoutCookie makes sure a callback that the app never started
// is refused, which is what a forged callback looks like.
func TestTakeLoginWithoutCookie(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true, 3600)

	_, err := store.takeLogin(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	is.True(err != nil) // a callback without a pending login should fail
}

// TestTakeLoginIsSingleUse makes sure a callback cannot be replayed.
func TestTakeLoginIsSingleUse(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true, 3600)
	saved := httptest.NewRecorder()
	is.NoErr(store.saveLogin(saved, httptest.NewRequest(http.MethodGet, "/", nil), login{State: "state-value"}))

	taken := httptest.NewRecorder()
	_, err := store.takeLogin(taken, replay(t, saved))
	is.NoErr(err)

	_, err = store.takeLogin(httptest.NewRecorder(), replay(t, taken))

	is.True(err != nil) // the login state must not survive its first use
}
