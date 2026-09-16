// Package utils provides HTTP response utilities for the Ekomasi e-commerce platform.
//
// This file contains HTTP response handling functionality:
//   - Standardized JSON response formatting
//   - Success and error response helpers
//   - Request/response logging (file and database)
//   - Sensitive data masking for security
//   - User context extraction
//   - Response payload validation
//
// Response Features:
//   - Consistent JSON structure (status_code, message, data)
//   - Automatic logging with user ID and timing
//   - Log level determination from HTTP status codes
//   - Request summary generation with sensitive field masking
//   - Empty payload detection and handling
//
// Logging:
//   - File logging with structured format
//   - Database logging with metadata
//   - User tracking (ID and role)
//   - Performance timing
//   - Module and function tracking
package utils

import (
	"bytes"
	"ekomasi_backend/logger"
	"ekomasi_backend/middleware"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CollectiveInfo contains metadata about the response.
//
// This struct groups module, description, and HTTP status code
// for consistent response handling and logging.
type CollectiveInfo struct {
	Module      string // Module name (e.g., "auth", "products", "orders")
	Description string // Operation description for logging
	Code        int    // HTTP status code (200, 400, 500, etc.)
}

// SuccessJSONResponseOptions configures successful JSON responses.
//
// Contains all data needed to generate a standardized success response
// with logging and performance tracking.
type SuccessJSONResponseOptions struct {
	CollectiveInfo CollectiveInfo // Response metadata
	Payload        interface{}    // Response data (can be nil, struct, array, etc.)
	Message        string         // Human-readable message
	TimeTaken      time.Duration  // Request processing duration
	Function       string         // Handler function name for logging
	Request        *http.Request  // Original HTTP request
	RawBody        string         // Request body summary for logging
}

// ErrorJSONResponseOptions configures error JSON responses.
//
// Similar to SuccessJSONResponseOptions but without payload field
// since error responses typically don't include data.
type ErrorJSONResponseOptions struct {
	CollectiveInfo CollectiveInfo // Response metadata
	Message        string         // Error message
	TimeTaken      time.Duration  // Request processing duration
	Function       string         // Handler function name for logging
	Request        *http.Request  // Original HTTP request
	RawBody        string         // Request body summary for logging
}

// RespondWithError sends a standardized error JSON response.
//
// This is a convenience wrapper around RespondWithJSON that converts
// error options to success options with nil payload. This ensures
// consistent error response format.
//
// Parameters:
//   - w: http.ResponseWriter - HTTP response writer
//   - erropts: ErrorJSONResponseOptions - Error response configuration
var RespondWithError = func(w http.ResponseWriter, erropts ErrorJSONResponseOptions) {

	// Convert error options to success options with nil payload
	RespondWithJSON(w, SuccessJSONResponseOptions{
		CollectiveInfo: erropts.CollectiveInfo,
		Payload:        nil, // Errors don't include data payload
		Message:        erropts.Message,
		TimeTaken:      erropts.TimeTaken,
		Function:       erropts.Function,
		Request:        erropts.Request,
		RawBody:        erropts.RawBody,
	})

}

// RespondWithGinError sends a standardized error JSON response using a Gin context.
// Preserves structured logging, execution metrics, and response formatting.
func RespondWithGinError(c *gin.Context, erropts ErrorJSONResponseOptions) {
	if erropts.Request == nil {
		erropts.Request = c.Request
	}
	RespondWithGinJSON(c, SuccessJSONResponseOptions{
		CollectiveInfo: erropts.CollectiveInfo,
		Payload:        nil,
		Message:        erropts.Message,
		TimeTaken:      erropts.TimeTaken,
		Function:       erropts.Function,
		Request:        erropts.Request,
		RawBody:        erropts.RawBody,
	})
}

// RespondWithGinJSON sends a standardized JSON response using a Gin context.
// Fully compatible with Gin framework while preserving file/database logging and timing.
func RespondWithGinJSON(c *gin.Context, opts SuccessJSONResponseOptions) {
	if opts.Request == nil {
		opts.Request = c.Request
	}

	ctx := opts.Request.Context()
	userID := "unknown"
	userRole := "customer"
	user, ok := middleware.UserFromContext(ctx)
	if ok {
		userID = user.ID
		userRole = user.Role
	}

	log.Printf(
		`[%s] [%s] User: %s | Request Info: %s | Function: %s | Time Taken: %s | Status Code: %d | Message: %s`,
		http.StatusText(opts.CollectiveInfo.Code),
		time.Now().Format("2006-01-02 15:04:05"),
		userID,
		opts.RawBody,
		opts.Function,
		opts.TimeTaken,
		opts.CollectiveInfo.Code,
		opts.Message,
	)

	logger.Log(logger.LogEntry{
		Level:   levelFromStatus(opts.CollectiveInfo.Code),
		Message: opts.Message,
		UserID:  &userID,
		Metadata: map[string]interface{}{
			"Request Info": opts.RawBody,
			"Function":     opts.Function,
			"Time Taken":   opts.TimeTaken.String(),
			"Module":       opts.CollectiveInfo.Module,
			"Description":  opts.CollectiveInfo.Description,
		},
		Module: &opts.CollectiveInfo.Module,
		Role:   &userRole,
	})

	response := map[string]interface{}{
		"status_code": opts.CollectiveInfo.Code,
		"message":     opts.Message,
	}

	if opts.CollectiveInfo.Code < 400 || (opts.Payload != nil && !isEmpty(opts.Payload)) {
		response["data"] = opts.Payload
	}

	c.JSON(opts.CollectiveInfo.Code, response)
}

// RespondWithJSON sends a standardized JSON response with logging.
//
// This is the main response function that:
// 1. Extracts user context (ID and role)
// 2. Logs to file with structured format
// 3. Logs to database with metadata
// 4. Generates JSON response
// 5. Sets appropriate headers and status code
//
// Response structure:
//
//	{"status_code": 200, "message": "...", "data": {...}}
//
// Data field is included only for:
//   - Success responses (status < 400)
//   - Non-empty payloads (checked with isEmpty)
//
// Parameters:
//   - w: http.ResponseWriter - HTTP response writer
//   - opts: SuccessJSONResponseOptions - Response configuration
var RespondWithJSON = func(w http.ResponseWriter, opts SuccessJSONResponseOptions) {
	ctx := opts.Request.Context()

	// Extract user information from context
	userID := "unknown"
	userRole := "customer" // Default role
	user, ok := middleware.UserFromContext(ctx)
	if ok {
		userID = user.ID
		userRole = user.Role
	}

	// Log to file with structured format
	log.Printf(
		`[%s] [%s] User: %s | Request Info: %s | Function: %s | Time Taken: %s | Status Code: %d | Message: %s`,
		http.StatusText(opts.CollectiveInfo.Code), // HTTP status text (e.g., "OK", "Bad Request")
		time.Now().Format("2006-01-02 15:04:05"),  // Timestamp
		userID,                                    // User performing action
		opts.RawBody,                              // Request summary
		opts.Function,                             // Handler function name
		opts.TimeTaken,                            // Processing duration
		opts.CollectiveInfo.Code,                  // HTTP status code
		opts.Message,                              // Response message
	)

	// Log to database with metadata
	logger.Log(logger.LogEntry{
		Level:   levelFromStatus(opts.CollectiveInfo.Code), // Determine log level from status code
		Message: opts.Message,
		UserID:  &userID,
		Metadata: map[string]interface{}{
			"Request Info": opts.RawBody,
			"Function":     opts.Function,
			"Time Taken":   opts.TimeTaken.String(),
			"Module":       opts.CollectiveInfo.Module,
			"Description":  opts.CollectiveInfo.Description,
		},
		Module: &opts.CollectiveInfo.Module,
		Role:   &userRole,
	})

	// Create standardized JSON response structure
	response := map[string]interface{}{
		"status_code": opts.CollectiveInfo.Code,
		"message":     opts.Message,
	}

	// Include data field only for success responses or non-empty payloads
	if opts.CollectiveInfo.Code < 400 || (opts.Payload != nil && !isEmpty(opts.Payload)) {
		response["data"] = opts.Payload
	}

	// Marshal response to JSON
	respBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	// Set response headers and write JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(opts.CollectiveInfo.Code) // Set HTTP status code
	w.Write(respBytes)                      // Write JSON response body
}

// isEmpty checks if a value is empty using reflection.
//
// Determines emptiness based on type:
//   - Slices/Arrays/Maps: Empty if length is 0
//   - Pointers/Interfaces: Empty if nil
//   - Other types: Not considered empty
//
// Used to decide whether to include payload in error responses.
//
// Parameters:
//   - x: interface{} - Value to check
//
// Returns:
//   - bool: true if empty, false otherwise
func isEmpty(x interface{}) bool {
	v := reflect.ValueOf(x)

	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return v.Len() == 0 // Empty if no elements
	case reflect.Ptr, reflect.Interface:
		return v.IsNil() // Empty if nil pointer
	default:
		return false // Other types not considered empty
	}
}

// levelFromStatus determines log level from HTTP status code.
//
// Log level mapping:
//   - 5xx (Server Error): ErrorLevel
//   - 4xx (Client Error): WarnLevel
//   - 2xx/3xx (Success/Redirect): InfoLevel
//
// Parameters:
//   - code: int - HTTP status code
//
// Returns:
//   - logger.LogLevel: Appropriate log level
func levelFromStatus(code int) logger.LogLevel {
	switch {
	case code >= 500:
		return logger.ErrorLevel // Server errors
	case code >= 400:
		return logger.WarnLevel // Client errors
	default:
		return logger.InfoLevel // Success responses
	}
}

// GetRequestSummary generates a formatted request summary for logging.
//
// This function:
// 1. Reads request body (and restores it for future use)
// 2. Masks sensitive fields (passwords, API keys, etc.)
// 3. Omits binary/multipart data
// 4. Formats summary with timestamp, method, path, address, payload
//
// Used for logging requests without exposing sensitive data.
//
// Parameters:
//   - r: *http.Request - HTTP request to summarize
//
// Returns:
//   - string: Formatted request summary with masked sensitive fields
var GetRequestSummary = func(r *http.Request) string {
	contentType := r.Header.Get("Content-Type")

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		bodyBytes = []byte("error reading body")
	}

	// Restore body for subsequent handlers
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var maskedPayload string
	// Don't log binary/multipart payloads (file uploads, images)
	if strings.HasPrefix(contentType, "multipart/form-data") ||
		strings.HasPrefix(contentType, "image/") {
		maskedPayload = "[binary data omitted]"
	} else {
		// Mask sensitive fields if JSON payload
		maskedPayload = maskSensitiveFields(bodyBytes)
	}

	// Return formatted summary
	return fmt.Sprintf(
		"Time: %s | Method: %s | Path: %s | Address: %s | Payload: %s",
		time.Now().Format("2006-01-02 15:04:05"),
		r.Method,
		r.URL.Path,
		r.RemoteAddr,
		maskedPayload,
	)
}

// maskSensitiveFields replaces sensitive field values with safe placeholders.
//
// Masks the following fields (case-insensitive):
//   - password: Replaced with "***************"
//   - apikey: Replaced with "***************"
//   - body: Replaced with "" (empty string)
//   - image: Replaced with "" (empty string)
//   - image_url: Replaced with "" (empty string)
//
// Non-JSON payloads are returned unchanged.
//
// Parameters:
//   - body: []byte - Raw request body
//
// Returns:
//   - string: JSON string with sensitive fields masked, or original if not JSON
func maskSensitiveFields(body []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return string(body) // Not JSON, return as-is
	}

	// Define sensitive fields and their masked values
	sensitiveKeys := map[string]string{
		"password":  "***************", // Hide passwords
		"apikey":    "***************", // Hide API keys
		"body":      "",                // Omit large body fields
		"image":     "",                // Omit image data
		"image_url": "",                // Omit image URLs
	}

	// Replace sensitive field values
	for key := range data {
		if masked, ok := sensitiveKeys[strings.ToLower(key)]; ok {
			data[key] = masked
		}
	}

	// Marshal back to JSON
	if maskedJSON, err := json.Marshal(data); err == nil {
		return string(maskedJSON)
	}
	return string(body) // Return original if marshaling fails
}
