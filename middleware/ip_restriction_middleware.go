package middleware

import (
	"ekomasi_backend/models"
	"log"
	"net/http"
	"os"
	"strings"
)

// getClientIP extracts the real client IP from the request
// Checks multiple headers in order of preference to get the actual client IP
func getClientIP(r *http.Request) string {
	// Check X-Real-IP header first (commonly set by proxies)
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	// Check X-Forwarded-For header (can contain multiple IPs)
	// The first IP is the original client
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check CF-Connecting-IP (Cloudflare)
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	// Check True-Client-IP (Akamai, Cloudflare)
	if ip := r.Header.Get("True-Client-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	// Fall back to RemoteAddr (strip port if present)
	ip := r.RemoteAddr
	if pos := strings.LastIndex(ip, ":"); pos != -1 {
		ip = ip[:pos]
	}
	return ip
}

// RestrictSwaggerAccess is middleware that limits access to Swagger documentation
// based on an allowed list of IPs in the database and a developer IP from environment variables.
func RestrictSwaggerAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract visitor's IP address
		ip := getClientIP(r)
		log.Printf("Visitor IP:::: %s", ip)
		// Allow access if IP matches developer IP from env
		devIP := os.Getenv("DEVELOPER_IP")
		if devIP != "" && ip == devIP {
			next.ServeHTTP(w, r)
			return
		}

		// Check database for allowed IPs
		isAllowed, err := models.IsIPAllowed(models.DB, ip)
		if err != nil {
			// Log error but deny access if we can't verify
			sendErrorResponse(w, http.StatusInternalServerError, "Internal server error during IP verification")
			return
		}

		if isAllowed {
			next.ServeHTTP(w, r)
			return
		}
		// Deny access
		sendErrorResponse(w, http.StatusForbidden, "Access to Swagger documentation is restricted.")
	})
}
