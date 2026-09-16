package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// SwaggerBasicAuth returns a Gin middleware for HTTP Basic Authentication
// protecting Swagger documentation using SWAGGER_USER and SWAGGER_PASSWORD env vars.
func SwaggerBasicAuth() gin.HandlerFunc {
	user := os.Getenv("SWAGGER_USER")
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv("SWAGGER_PASSWORD")
	if pass == "" {
		pass = "EkomasiSecure2026!"
	}

	return gin.BasicAuth(gin.Accounts{
		user: pass,
	})
}
