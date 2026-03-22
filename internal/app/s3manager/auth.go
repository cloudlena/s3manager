package s3manager

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

const sessionCookieName = "s3manager_session"

// generateSessionToken creates an HMAC-SHA256 token from username, password, and a secret.
func generateSessionToken(username, password, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(username + ":" + password))
	return hex.EncodeToString(mac.Sum(nil))
}

// AuthMiddleware checks for a valid session cookie. If the cookie is missing or invalid,
// it redirects the user to the login page. Paths like /login, /api/login, /logout, and
// /static/ are exempted so the login page can render properly.
func AuthMiddleware(username, password, sessionSecret, rootURL string) func(http.Handler) http.Handler {
	expectedToken := generateSessionToken(username, password, sessionSecret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Exempt paths that must work without auth
			if path == rootURL+"/login" ||
				path == rootURL+"/api/login" ||
				path == rootURL+"/logout" ||
				strings.HasPrefix(path, rootURL+"/static/") {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value != expectedToken {
				// Redirect to login, preserving the original URL
				redirectTo := r.URL.RequestURI()
				http.Redirect(w, r, rootURL+"/login?redirect="+redirectTo, http.StatusFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HandleLoginPage renders the login page template.
func HandleLoginPage(templates fs.FS, rootURL string) http.HandlerFunc {
	type loginData struct {
		RootURL  string
		Redirect string
		Error    bool
	}

	return func(w http.ResponseWriter, r *http.Request) {
		redirect := r.URL.Query().Get("redirect")
		if redirect == "" {
			redirect = rootURL + "/buckets"
		}
		errorParam := r.URL.Query().Get("error")

		data := loginData{
			RootURL:  rootURL,
			Redirect: redirect,
			Error:    errorParam == "1",
		}

		t, err := template.ParseFS(templates, "login.html.tmpl")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = t.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// HandleLoginSubmit validates the submitted credentials and sets a session cookie on success.
func HandleLoginSubmit(username, password, sessionSecret, rootURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		submittedUser := r.FormValue("username")
		submittedPass := r.FormValue("password")
		redirect := r.FormValue("redirect")
		if redirect == "" {
			redirect = rootURL + "/buckets"
		}

		if submittedUser != username || submittedPass != password {
			http.Redirect(w, r, rootURL+"/login?error=1&redirect="+redirect, http.StatusFound)
			return
		}

		token := generateSessionToken(username, password, sessionSecret)
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, redirect, http.StatusFound)
	}
}

// HandleLogout clears the session cookie and redirects to the login page.
func HandleLogout(rootURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		http.Redirect(w, r, rootURL+"/login", http.StatusFound)
	}
}

