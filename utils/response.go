package utils

import (
	"adenzo_backend/logger"
	"adenzo_backend/middleware"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type CollectiveInfo struct {
	Module      string
	Description string
	Code        int
}
type SuccessJSONResponseOptions struct {
	CollectiveInfo CollectiveInfo
	Payload        interface{}
	Message        string
	TimeTaken      time.Duration
	Function       string
	Request        *http.Request
	RawBody        string
}
type ErrorJSONResponseOptions struct {
	CollectiveInfo CollectiveInfo
	Message        string
	TimeTaken      time.Duration
	Function       string
	Request        *http.Request
	RawBody        string
}

var RespondWithError = func(w http.ResponseWriter, erropts ErrorJSONResponseOptions) {

	RespondWithJSON(w, SuccessJSONResponseOptions{
		CollectiveInfo: erropts.CollectiveInfo,
		Payload:        nil,
		Message:        erropts.Message,
		TimeTaken:      erropts.TimeTaken,
		Function:       erropts.Function,
		Request:        erropts.Request,
		RawBody:        erropts.RawBody,
	})

}

var RespondWithJSON = func(w http.ResponseWriter, opts SuccessJSONResponseOptions) {
	ctx := opts.Request.Context()

	userID := "unknown"
	user, ok := middleware.UserFromContext(ctx)
	if ok {
		userID = user.ID
	}
	// Log to file
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

	// Log to DB
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
	})

	// Create response
	response := map[string]interface{}{
		"status_code": opts.CollectiveInfo,
		"message":     opts.Message,
	}

	if opts.CollectiveInfo.Code < 400 || (opts.Payload != nil && !isEmpty(opts.Payload)) {
		response["data"] = opts.Payload
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(opts.CollectiveInfo.Code)
	w.Write(respBytes)
}

func isEmpty(x interface{}) bool {
	v := reflect.ValueOf(x)

	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// choose level level based on the code
func levelFromStatus(code int) logger.LogLevel {
	switch {
	case code >= 500:
		return logger.ErrorLevel
	case code >= 400:
		return logger.WarnLevel
	default:
		return logger.InfoLevel
	}
}

// GetRequestSummary returns a formatted string with method, path, address, and body
var GetRequestSummary = func(r *http.Request) string {
	contentType := r.Header.Get("Content-Type")
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		bodyBytes = []byte("error reading body")
	}

	// Restore body for future use
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	var maskedPayload string
	// Don't log binary/multipart payloads
	if strings.HasPrefix(contentType, "multipart/form-data") ||
		strings.HasPrefix(contentType, "image/") {
		maskedPayload = "[binary data omitted]"
	} else {
		// Mask sensitive fields if JSON
		maskedPayload = maskSensitiveFields(bodyBytes)
	}

	return fmt.Sprintf(
		"Time: %s | Method: %s | Path: %s | Address: %s | Payload: %s",
		time.Now().Format("2006-01-02 15:04:05"),
		r.Method,
		r.URL.Path,
		r.RemoteAddr,
		maskedPayload,
	)
}

// maskSensitiveFields replaces sensitive keys with safe values
func maskSensitiveFields(body []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return string(body) // not JSON, return raw
	}

	sensitiveKeys := map[string]string{
		"password":  "***************",
		"apikey":    "***************",
		"body":      "",
		"image":     "",
		"image_url": "",
	}

	for key := range data {
		if masked, ok := sensitiveKeys[strings.ToLower(key)]; ok {
			data[key] = masked
		}
	}

	if maskedJSON, err := json.Marshal(data); err == nil {
		return string(maskedJSON)
	}
	return string(body)
}
