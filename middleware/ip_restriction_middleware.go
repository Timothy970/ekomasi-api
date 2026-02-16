package middleware

import (
	"adenzo_backend/models"
	"log"
	"net/http"
	"os"
	"strings"
)

// RestrictSwaggerAccess is middleware that limits access to Swagger documentation
// based on an allowed list of IPs in the database and a developer IP from environment variables.
func RestrictSwaggerAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract visitor's IP address
		// RemoteAddr usually contains "ip:port"
		ip := r.RemoteAddr
		if pos := strings.LastIndex(ip, ":"); pos != -1 {
			ip = ip[:pos]
		}

		// Check for X-Forwarded-For if behind a proxy
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			ip = strings.TrimSpace(ips[0])
		}
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
