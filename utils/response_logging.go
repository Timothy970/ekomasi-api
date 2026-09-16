package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

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
	var data map[string]any
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
