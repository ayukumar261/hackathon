package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ctxKey struct{}

var userKey = ctxKey{}

func RequireUser(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("session_id")
			if err != nil || c.Value == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			sid, err := uuid.Parse(c.Value)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var session models.Session
			if err := db.First(&session, "id = ?", sid).Error; err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if time.Now().After(session.ExpiresAt) {
				db.Delete(&session)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var user models.User
			if err := db.First(&user, "id = ?", session.UserID).Error; err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFrom(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(userKey).(*models.User)
	return u, ok
}
