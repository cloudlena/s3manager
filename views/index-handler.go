package views

import "net/http"

// IndexHandler forwards to "/buckets"
func IndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/buckets", http.StatusPermanentRedirect)
	})
}
