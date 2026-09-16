// Package middleware provides HTTP middleware components for request processing.
// Implements JWT-based authentication with token validation, blacklist checking,
// and user context propagation throughout the request lifecycle.
// Supports both access tokens and refresh tokens with type validation.
package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"ekomasi_backend/dtos"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// bearer is the expected prefix for Authorization header values
var (
	bearer                = "Bearer "
	tokenMissingMsg       = "Unauthorized access"
	invalidTokenMsg       = "Invalid or expired token"
	invalidTokenClaimsMsg = "Invalid token claims"
	blacklist             = "blacklist:"
)

// GinAuthenticateToken is native Gin middleware that validates JWT access tokens for protected routes.
func GinAuthenticateToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		r := c.Request
		authHeader := r.Header.Get("Authorization")
		tokenString := ""
		if strings.HasPrefix(authHeader, bearer) {
			tokenString = strings.TrimPrefix(authHeader, bearer)
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status_code": http.StatusUnauthorized,
				"message":     tokenMissingMsg,
			})
			c.Abort()
			return
		}

		if dtos.Redis != nil {
			val, err := dtos.Redis.Get(r.Context(), blacklist+tokenString).Result()
			if err == nil && val == "1" {
				c.JSON(http.StatusForbidden, gin.H{
					"status_code": http.StatusForbidden,
					"message":     "Token is invalidated",
				})
				c.Abort()
				return
			}
		}

		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "secret"
		}

		token, err := jwt.ParseWithClaims(tokenString, &dtos.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusForbidden, gin.H{
				"status_code": http.StatusForbidden,
				"message":     invalidTokenMsg,
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*dtos.CustomClaims)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"status_code": http.StatusForbidden,
				"message":     invalidTokenClaimsMsg,
			})
			c.Abort()
			return
		}

		user := AuthenticatedUser{
			ID:          claims.UserID,
			Email:       claims.Email,
			FirstName:   claims.FirstName,
			LastName:    claims.LastName,
			Role:        claims.Role,
			Phone:       claims.Phone,
			Permissions: claims.Permissions,
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)
		ctx := context.WithValue(r.Context(), userContextKey, user)
		c.Request = r.WithContext(ctx)

		c.Next()
	}
}

// AuthenticateToken is middleware that validates JWT access tokens for protected routes.
// Extracts token from Authorization header, validates against blacklist and JWT signature,
// and injects authenticated user information into request context.
// Returns 401 for missing tokens, 403 for invalid/blacklisted tokens.
func AuthenticateToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract Authorization header and parse Bearer token
		authHeader := r.Header.Get("Authorization")
		tokenString := ""
		if strings.HasPrefix(authHeader, bearer) {
			// Remove "Bearer " prefix to get raw JWT
			tokenString = strings.TrimPrefix(authHeader, bearer)
		}

		if tokenString == "" {
			// No token provided - authentication required
			response := map[string]interface{}{
				"status_code": http.StatusUnauthorized,
				"message":     tokenMissingMsg,
			}
			sendResponse(response, http.StatusUnauthorized, w)
			return
		}
		// Check if token has been blacklisted (e.g., after logout)
		val, err := dtos.Redis.Get(r.Context(), blacklist+tokenString).Result()
		if err == nil && val == "1" {
			// Token exists in blacklist - deny access
			sendErrorResponse(w, http.StatusForbidden, "Token is invalidated")
			return
		}
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			log.Println("JWT_SECRET environment variable is not set")
			sendErrorResponse(w, http.StatusInternalServerError, "Server configuration error")
			return
		}
		// Parse and validate JWT signature and expiration
		token, err := jwt.ParseWithClaims(tokenString, &dtos.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Return secret key for signature validation
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			// Token parsing failed or signature invalid or expired
			response := map[string]interface{}{
				"status_code": http.StatusForbidden,
				"message":     invalidTokenMsg,
			}
			sendResponse(response, http.StatusForbidden, w)
			return
		}

		// Extract custom claims from validated token
		claims, ok := token.Claims.(*dtos.CustomClaims)
		if !ok {
			// Claims structure doesn't match expected format
			sendErrorResponse(w, http.StatusForbidden, invalidTokenClaimsMsg)
			return
		}
		if ok {
			// Build authenticated user object from token claims
			user := AuthenticatedUser{
				ID:          claims.UserID,
				Email:       claims.Email,
				FirstName:   claims.FirstName,
				LastName:    claims.LastName,
				Role:        claims.Role,
				Phone:       claims.Phone,
				Permissions: claims.Permissions,
			}
			// Inject user into request context for downstream handlers
			ctx := contextWithUser(r.Context(), user)
			r = r.WithContext(ctx)
		}
		log.Printf("next after auth")
		// Token valid - proceed to next handler
		next.ServeHTTP(w, r)
	})
}

// AuthenticateRefreshToken is middleware that validates JWT refresh tokens.
// Similar to AuthenticateToken but specifically validates refresh token type.
// Used for token refresh endpoints to ensure only refresh tokens are accepted.
// Returns 401 for missing tokens, 403 for invalid tokens or wrong token type.
func AuthenticateRefreshToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract Authorization header and parse Bearer token
		authHeader := r.Header.Get("Authorization")
		tokenString := ""
		if strings.HasPrefix(authHeader, bearer) {
			// Remove "Bearer " prefix to get raw JWT
			tokenString = strings.TrimPrefix(authHeader, bearer)
		}

		if tokenString == "" {
			// No token provided
			sendErrorResponse(w, http.StatusUnauthorized, tokenMissingMsg)
			return
		}

		// Check if token has been blacklisted (e.g., after logout or refresh)
		val, err := dtos.Redis.Get(r.Context(), blacklist+tokenString).Result()
		if err == nil && val == "1" {
			// Token exists in blacklist - deny access
			sendErrorResponse(w, http.StatusForbidden, "Token has been invalidated")
			return
		}

		// Parse and validate JWT signature and expiration
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Return secret key for signature validation
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil || !token.Valid {
			// Token parsing failed or signature invalid or expired
			sendErrorResponse(w, http.StatusForbidden, invalidTokenMsg)
			return
		}

		// Extract standard JWT claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			// Claims structure doesn't match expected format
			sendErrorResponse(w, http.StatusForbidden, invalidTokenClaimsMsg)
			return
		}

		// Validate token type to ensure it's a refresh token
		if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh_token" {
			// Wrong token type - access tokens not allowed on refresh endpoints
			sendErrorResponse(w, http.StatusForbidden, "Invalid token type")
			return
		}

		var permissions []string
		if perms, ok := claims["permissions"].([]interface{}); ok {
			for _, perm := range perms {
				if permStr, ok := perm.(string); ok {
					permissions = append(permissions, permStr)
				}
			}
		}

		// Build authenticated user object from token claims
		user := AuthenticatedUser{
			ID:          fmt.Sprintf("%v", claims["id"]),
			Email:       fmt.Sprintf("%v", claims["email"]),
			FirstName:   fmt.Sprintf("%v", claims["first_name"]),
			LastName:    fmt.Sprintf("%v", claims["last_name"]),
			Role:        fmt.Sprintf("%v", claims["role"]),
			Phone:       fmt.Sprintf("%v", claims["phone_number"]),
			Permissions: permissions,
		}
		// Inject user into request context for downstream handlers
		ctx := contextWithUser(r.Context(), user)
		r = r.WithContext(ctx)

		// Refresh token valid - proceed to next handler
		next.ServeHTTP(w, r)
	})
}

// AuthenticatedUser represents a verified user extracted from a validated JWT.
