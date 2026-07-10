package admin

import (
	"crypto/subtle"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// BasicAuth enforces a single admin credential (bcrypt hash from config)
// over every wrapped route. Intended to ride behind Caddy's TLS
// termination, appropriate for a single-operator back office.
func BasicAuth(username, passHash string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 {
				unauthorized(w)
				return
			}
			if err := bcrypt.CompareHashAndPassword([]byte(passHash), []byte(pass)); err != nil {
				unauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="qrsurvey admin"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}
