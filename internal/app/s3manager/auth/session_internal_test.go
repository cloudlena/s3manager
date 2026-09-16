package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

	store := newSessionStore(deriveSessionKey("secret"), "/", true)
	identity := &Identity{
		Subject: "abc-123",
		Name:    "Jane Doe",
		Email:   "jane@example.com",
		Groups:  []string{"s3-admins"},
		Role:    RoleWriter,
	}

	resp := httptest.NewRecorder()
	is.NoErr(store.saveIdentity(resp, httptest.NewRequest(http.MethodGet, "/", nil), identity, 3600))

	is.Equal(store.identity(replay(t, resp)), identity)
}

func TestSessionWithoutCookie(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true)

	is.Equal(store.identity(httptest.NewRequest(http.MethodGet, "/", nil)), (*Identity)(nil))
}

// TestSessionRejectsForeignKey makes sure a cookie sealed with another key is
// not accepted, which is what stops a user from forging their own role.
func TestSessionRejectsForeignKey(t *testing.T) {
	is := is.New(t)

	resp := httptest.NewRecorder()
	writer := newSessionStore(deriveSessionKey("one secret"), "/", true)
	is.NoErr(writer.saveIdentity(resp, httptest.NewRequest(http.MethodGet, "/", nil), &Identity{Subject: "abc", Role: RoleWriter}, 3600))

	reader := newSessionStore(deriveSessionKey("another secret"), "/", true)

	is.Equal(reader.identity(replay(t, resp)), (*Identity)(nil))
}

func TestSessionCookieAttributes(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/s3", true)
	resp := httptest.NewRecorder()
	is.NoErr(store.saveIdentity(resp, httptest.NewRequest(http.MethodGet, "/", nil), &Identity{Subject: "abc", Role: RoleViewer}, 3600))

	cookie := resp.Result().Cookies()[0]

	is.Equal(cookie.Name, userSessionName)
	is.Equal(cookie.Path, "/s3")
	is.True(cookie.HttpOnly)                        // the cookie is never read from JavaScript
	is.True(cookie.Secure)                          // the cookie is only sent over HTTPS
	is.Equal(cookie.SameSite, http.SameSiteLaxMode) // strict would break the redirect back from the provider
}

func TestClearIdentity(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true)
	saved := httptest.NewRecorder()
	is.NoErr(store.saveIdentity(saved, httptest.NewRequest(http.MethodGet, "/", nil), &Identity{Subject: "abc", Role: RoleWriter}, 3600))

	cleared := httptest.NewRecorder()
	is.NoErr(store.clearIdentity(cleared, replay(t, saved)))

	is.Equal(store.identity(replay(t, cleared)), (*Identity)(nil))
}

func TestLoginRoundTrip(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true)
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

	store := newSessionStore(deriveSessionKey("secret"), "/", true)

	_, err := store.takeLogin(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	is.True(err != nil) // a callback without a pending login should fail
}

// TestTakeLoginIsSingleUse makes sure a callback cannot be replayed.
func TestTakeLoginIsSingleUse(t *testing.T) {
	is := is.New(t)

	store := newSessionStore(deriveSessionKey("secret"), "/", true)
	saved := httptest.NewRecorder()
	is.NoErr(store.saveLogin(saved, httptest.NewRequest(http.MethodGet, "/", nil), login{State: "state-value"}))

	taken := httptest.NewRecorder()
	_, err := store.takeLogin(taken, replay(t, saved))
	is.NoErr(err)

	_, err = store.takeLogin(httptest.NewRecorder(), replay(t, taken))

	is.True(err != nil) // the login state must not survive its first use
}
