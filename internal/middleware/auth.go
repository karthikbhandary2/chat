package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/karthikbhandary2/chat/internal/auth"
)

type contextKey string
const userIDKey contextKey = "user_id"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		id, err := auth.ValidateToken(tokenString)
		if err != nil {
			log.Printf("error validating token: %v", err)
			http.Error(w, "validation error", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}