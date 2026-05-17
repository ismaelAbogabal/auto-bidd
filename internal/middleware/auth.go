package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/ismaelfi/auto-bidd/internal/models"
	"gorm.io/gorm"
)

type contextKey string

const UserContextKey contextKey = "user"

func GetUser(r *http.Request) *models.User {
	user, _ := r.Context().Value(UserContextKey).(*models.User)
	return user
}

func RequireAuth(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			var session models.Session
			err = db.Where("token = ? AND expires_at > ?", cookie.Value, time.Now()).First(&session).Error
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			var user models.User
			err = db.First(&user, "id = ?", session.UserID).Error
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}
