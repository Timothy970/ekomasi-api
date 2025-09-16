package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"adenzo_backend/dtos"

	"github.com/golang-jwt/jwt/v5"
)

var bearer = "Bearer "

func AuthenticateToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		tokenString := ""
		if strings.HasPrefix(authHeader, bearer) {
			tokenString = strings.TrimPrefix(authHeader, bearer)
		}

		if tokenString == "" {
			response := map[string]interface{}{
				"status_code": http.StatusUnauthorized,
				"message":     "Token missing",
			}
			sendResponse(response, http.StatusUnauthorized, w)
			return
		}
		// Check if token is blacklisted
		val, err := dtos.Redis.Get(r.Context(), "blacklist:"+tokenString).Result()
		if err == nil && val == "1" {
			sendErrorResponse(w, http.StatusForbidden, "Token has been invalidated")
			return
		}
		// parse and validate JWT
		token, err := jwt.ParseWithClaims(tokenString, &dtos.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil || !token.Valid {
			response := map[string]interface{}{
				"status_code": http.StatusForbidden,
				"message":     "Invalid or expired token",
			}
			sendResponse(response, http.StatusForbidden, w)
			return
		}

		// set user info in context (optional)
		claims, ok := token.Claims.(*dtos.CustomClaims)
		if !ok {
			sendErrorResponse(w, http.StatusForbidden, "Invalid token claims")
			return
		}
		if ok {
			user := AuthenticatedUser{
				ID:        claims.UserID,
				Email:     claims.Email,
				FirstName: claims.FirstName,
				LastName:  claims.LastName,
				Role:      claims.Role,
				Phone:     claims.Phone,
			}
			ctx := contextWithUser(r.Context(), user)
			r = r.WithContext(ctx)
		}
		log.Printf("next after auth")
		next.ServeHTTP(w, r)
	})
}
func AuthenticateRefreshToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		tokenString := ""
		if strings.HasPrefix(authHeader, bearer) {
			tokenString = strings.TrimPrefix(authHeader, bearer)
		}

		if tokenString == "" {
			sendErrorResponse(w, http.StatusUnauthorized, "Token missing")
			return
		}

		// Check if token is blacklisted
		val, err := dtos.Redis.Get(r.Context(), "blacklist:"+tokenString).Result()
		if err == nil && val == "1" {
			sendErrorResponse(w, http.StatusForbidden, "Token has been invalidated")
			return
		}

		// Parse and validate JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil || !token.Valid {
			sendErrorResponse(w, http.StatusForbidden, "Invalid or expired token")
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			sendErrorResponse(w, http.StatusForbidden, "Invalid token claims")
			return
		}

		// Ensure it's a refresh token
		if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh_token" {
			sendErrorResponse(w, http.StatusForbidden, "Invalid token type")
			return
		}

		// (Optional) attach claims to context
		user := AuthenticatedUser{
			ID:        fmt.Sprintf("%v", claims["id"]),
			Email:     fmt.Sprintf("%v", claims["email"]),
			FirstName: fmt.Sprintf("%v", claims["first_name"]),
			LastName:  fmt.Sprintf("%v", claims["last_name"]),
			Role:      fmt.Sprintf("%v", claims["role"]),
			Phone:     fmt.Sprintf("%v", claims["phone_number"]),
		}
		ctx := contextWithUser(r.Context(), user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

type AuthenticatedUser struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	Role      string
	Phone     string
}
type contextKey string

// const userIDKey contextKey = "userID"
const userContextKey contextKey = "authenticatedUser"

func contextWithUser(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	val := ctx.Value(userContextKey)

	if val == nil {
		log.Println("UserFromContext: no value found in context for userContextKey")
		return AuthenticatedUser{}, false
	}

	user, ok := val.(AuthenticatedUser)
	if !ok {
		log.Printf("UserFromContext: value found, but wrong type: %T (expected AuthenticatedUser)\n", val)
		return AuthenticatedUser{}, false
	}

	return user, true
}

func sendResponse(response map[string]interface{}, code int, w http.ResponseWriter) {
	respBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(respBytes)
}
func sendErrorResponse(w http.ResponseWriter, code int, message string) {
	response := map[string]interface{}{
		"status_code": code,
		"message":     message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

//Helper function to delete/blacklist tok
