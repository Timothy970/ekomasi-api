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

type SuccessJSONResponseOptions struct {
	Code      int
	Payload   interface{}
	Message   string
	TimeTaken time.Duration
	Function  string
	Request   *http.Request
	RawBody   string
}
type ErrorJSONResponseOptions struct {
	Code      int
	Message   string
	TimeTaken time.Duration
	Function  string
	Request   *http.Request
	RawBody   string
}

func RespondWithError(w http.ResponseWriter, erropts ErrorJSONResponseOptions) {

	RespondWithJSON(w, SuccessJSONResponseOptions{
		Code:      erropts.Code,
		Payload:   nil,
		Message:   erropts.Message,
		TimeTaken: erropts.TimeTaken,
		Function:  erropts.Function,
		Request:   erropts.Request,
		RawBody:   erropts.RawBody,
	})

}

func RespondWithJSON(w http.ResponseWriter, opts SuccessJSONResponseOptions) {
	ctx := opts.Request.Context()

	userID := "unknown"
	user, ok := middleware.UserFromContext(ctx)
	if ok {
		userID = user.ID
	}
	// Log to file
	log.Printf(
		`[%s] [%s] User: %s | Request Info: %s | Function: %s | Time Taken: %s | Status Code: %d | Message: %s`,
		http.StatusText(opts.Code),
		time.Now().Format("2006-01-02 15:04:05"),
		userID,
		opts.RawBody,
		opts.Function,
		opts.TimeTaken,
		opts.Code,
		opts.Message,
	)

	// Log to DB
	logger.Log(logger.LogEntry{
		Level:   levelFromStatus(opts.Code),
		Message: opts.Message,
		UserID:  &userID,
		Metadata: map[string]interface{}{
			"Request Info": opts.RawBody,
			"Function":     opts.Function,
			"Time Taken":   opts.TimeTaken.String(),
		},
	})

	// Create response
	response := map[string]interface{}{
		"status_code": opts.Code,
		"message":     opts.Message,
	}

	if opts.Code < 400 || (opts.Payload != nil && !isEmpty(opts.Payload)) {
		response["data"] = opts.Payload
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(opts.Code)
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
func GetRequestSummary(r *http.Request) string {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		bodyBytes = []byte("error reading body")
	}

	// Restore body for future use
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Attempt to mask password field if body is JSON
	maskedPayload := string(bodyBytes)
	var data map[string]interface{}
	if json.Unmarshal(bodyBytes, &data) == nil {
		for key := range data {
			if strings.EqualFold(key, "password") {
				data[key] = "***************"
			}
		}
		if maskedJSON, err := json.Marshal(data); err == nil {
			maskedPayload = string(maskedJSON)
		}
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
