// Package middleware provides HTTP middleware components for request processing.
// Implements JWT-based authentication with token validation, blacklist checking,
// and user context propagation throughout the request lifecycle.
// Supports both access tokens and refresh tokens with type validation.
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

// bearer is the expected prefix for Authorization header values
var (
	bearer                = "Bearer "
	tokenMissingMsg       = "Unauthorized access"
	invalidTokenMsg       = "Invalid or expired token"
	invalidTokenClaimsMsg = "Invalid token claims"
	blacklist             = "blacklist:"
)

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
// Contains essential user identity and role information for authorization checks.
// Injected into request context after successful token validation.
type AuthenticatedUser struct {
	ID          string   // Unique user identifier from database
	Email       string   // User's email address
	FirstName   string   // User's first name
	LastName    string   // User's last name
	Role        string   // User's role (e.g., "admin", "user", "customer")
	Phone       string   // User's phone number
	Permissions []string // User's permissions
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// userContextKey is the key used to store/retrieve AuthenticatedUser from request context
const userContextKey contextKey = "authenticatedUser"

// contextWithUser creates a new context with the authenticated user attached.
// Used by middleware to propagate user information to downstream handlers.
func contextWithUser(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the authenticated user from request context.
// Returns (user, true) if user exists, (empty, false) if not found or wrong type.
// Used by handlers to access authenticated user information.
func UserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	// Attempt to retrieve value from context
	val := ctx.Value(userContextKey)

	if val == nil {
		// No user in context - likely unauthenticated request
		log.Println("UserFromContext: no value found in context for userContextKey")
		return AuthenticatedUser{}, false
	}

	// Type assert to AuthenticatedUser struct
	user, ok := val.(AuthenticatedUser)
	if !ok {
		// Value exists but wrong type - shouldn't happen in normal flow
		log.Printf("UserFromContext: value found, but wrong type: %T (expected AuthenticatedUser)\n", val)
		return AuthenticatedUser{}, false
	}

	return user, true
}

// sendResponse marshals and sends a JSON response with specified status code.
// Generic helper for sending structured JSON responses.
func sendResponse(response map[string]interface{}, code int, w http.ResponseWriter) {
	// Marshal response map to JSON bytes
	respBytes, err := json.Marshal(response)
	if err != nil {
		// JSON encoding failed - send plain text error
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	// Set JSON content type and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(respBytes)
}

// sendErrorResponse sends a standardized error response with status code and message.
// Convenience wrapper for consistent error response format.
func sendErrorResponse(w http.ResponseWriter, code int, message string) {
	// Build error response structure
	response := map[string]interface{}{
		"status_code": code,
		"message":     message,
	}
	// Set JSON content type and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	// Encode and send response
	json.NewEncoder(w).Encode(response)
}

// IsUserTokenPassed checks if a valid JWT token is present without full validation.
// Parses token without verifying signature or expiration for lightweight checks.
// Returns (user, true) if token present and parseable, (nil, false) otherwise.
// Useful for optional authentication scenarios or preliminary token checks.
func IsUserTokenPassed(r *http.Request) (*AuthenticatedUser, bool) {
	// Extract Authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, bearer) {
		// No Bearer token present
		return nil, false
	}

	// Remove "Bearer " prefix to get raw JWT
	tokenStr := strings.TrimPrefix(authHeader, bearer)

	// Parse token without signature verification (unverified parse)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		// Token malformed or unparseable
		return nil, false
	}

	// Extract claims from unverified token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		// Claims structure doesn't match expected format
		return nil, false
	}

	var permissions []string
	if perms, ok := claims["permissions"].([]interface{}); ok {
		for _, perm := range perms {
			if permStr, ok := perm.(string); ok {
				permissions = append(permissions, permStr)
			}
		}
	}

	// Build user object from token claims
	user := &AuthenticatedUser{
		ID:          fmt.Sprintf("%v", claims["id"]),
		Email:       fmt.Sprintf("%v", claims["email"]),
		FirstName:   fmt.Sprintf("%v", claims["first_name"]),
		LastName:    fmt.Sprintf("%v", claims["last_name"]),
		Role:        fmt.Sprintf("%v", claims["role"]),
		Phone:       fmt.Sprintf("%v", claims["phone_number"]),
		Permissions: permissions,
	}

	return user, true
}

// GetTokenAndAuthenticatedUser validates JWT access token and returns user info.
// Similar to AuthenticateToken middleware but returns values instead of continuing chain.
// Performs full validation including blacklist check, signature verification, and type check.
// Returns (true, user) on success, (false, nil) on any validation failure.
// Used by handlers that need explicit token validation with user extraction.
func GetTokenAndAuthenticatedUser(w http.ResponseWriter, r *http.Request) (bool, *AuthenticatedUser) {
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
		return false, nil
	}

	// Check if token has been blacklisted (e.g., after logout)
	val, err := dtos.Redis.Get(r.Context(), blacklist+tokenString).Result()
	if err == nil && val == "1" {
		// Token exists in blacklist - deny access
		sendErrorResponse(w, http.StatusForbidden, "Token has been invalidated")
		return false, nil
	}

	// Parse and validate JWT signature and expiration
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Return secret key for signature validation
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		// Token parsing failed or signature invalid or expired
		sendErrorResponse(w, http.StatusForbidden, invalidTokenMsg)
		return false, nil
	}

	// Extract standard JWT claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		// Claims structure doesn't match expected format
		sendErrorResponse(w, http.StatusForbidden, invalidTokenClaimsMsg)
		return false, nil
	}

	// Validate token type to ensure it's an access token (not refresh)
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "auth" {
		// Wrong token type - refresh tokens not allowed
		sendErrorResponse(w, http.StatusForbidden, "Invalid token type")
		return false, nil
	}

	// Build authenticated user object from validated token claims
	var permissions []string
	if perms, ok := claims["permissions"].([]interface{}); ok {
		for _, perm := range perms {
			if permStr, ok := perm.(string); ok {
				permissions = append(permissions, permStr)
			}
		}
	}

	user := AuthenticatedUser{
		ID:          fmt.Sprintf("%v", claims["id"]),
		Email:       fmt.Sprintf("%v", claims["email"]),
		FirstName:   fmt.Sprintf("%v", claims["first_name"]),
		LastName:    fmt.Sprintf("%v", claims["last_name"]),
		Role:        fmt.Sprintf("%v", claims["role"]),
		Phone:       fmt.Sprintf("%v", claims["phone_number"]),
		Permissions: permissions,
	}
	// Return success with authenticated user
	return true, &user
}
