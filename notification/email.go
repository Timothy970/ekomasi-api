// Package notification provides email notification services for the Ekomasi e-commerce platform.
//
// This package handles:
//   - Email sending via external API (V2 email service)
//   - Email payload construction with authentication
//   - Single email delivery
//   - Email scheduling support
//   - Attachment handling
//   - HTTP client communication with email service
//
// Environment variables required:
//   - APIKEY: API key for email service authentication
//   - PARTNERID: Partner ID for email service
//   - SENDER_EMAIL: From address for outgoing emails
//   - V2_URL: Base URL for the V2 email service API
package notification

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// SendEmail sends an email to the specified recipient.
//
// This is the primary email sending function. It constructs the email payload
// with authentication credentials from environment variables and sends it via
// the V2 email service API.
//
// Parameters:
//   - to: string - Recipient email address
//   - subject: string - Email subject line
//   - body: string - Email body content (can be HTML or plain text)
//
// Returns:
//   - error: API communication error, marshaling error, or nil on success
func SendEmail(to string, subject string, body string) error {
	// Construct email payload with authentication and metadata
	newPayload := map[string]interface{}{
		"apikey":       os.Getenv("APIKEY"),             // API authentication key
		"partnerID":    os.Getenv("PARTNERID"),          // Partner identification
		"from_address": os.Getenv("SENDER_EMAIL"),       // Sender email address
		"to_address":   to,                              // Recipient email address
		"body":         body,                            // Email content
		"subject":      subject,                         // Email subject
		"time":         time.Now(),                      // Current timestamp
		"date":         time.Now().Format("2006-01-02"), // Current date
		"scheduled":    false,                           // Immediate send (not scheduled)
		"attachments":  []interface{}{},                 // No attachments by default
	}

	// Send email via API and get response
	responseBody, err := sendSingleEmail(newPayload)
	if err != nil {
		log.Printf("Error sending email: %v", err)
	}
	log.Printf("Email sent to %s with subject '%s'. Response: %s", to, subject, responseBody)
	return err
}

// sendSingleEmail sends a single email via the V2 email service API.
//
// This internal function handles the HTTP communication with the email service.
// It marshals the payload to JSON, creates an HTTP POST request, and returns
// the response from the email service.
//
// Parameters:
//   - data: map[string]interface{} - Email payload containing all required fields
//     (apikey, partnerID, from_address, to_address, subject, body, etc.)
//
// Returns:
//   - string: Response body from the email service API
//   - error: JSON marshaling error, HTTP request error, or nil on success
func sendSingleEmail(data map[string]interface{}) (string, error) {
	// Get email service base URL from environment
	sendUrl := os.Getenv("V2_URL")
	url := sendUrl + "/services/send-email"
	method := "POST"

	// Marshal payload data to JSON
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return "", err
	}

	// Create JSON reader for request body
	payload := strings.NewReader(string(payloadBytes))

	// Initialize HTTP client
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return "", err
	}

	// Set content type header for JSON payload
	req.Header.Add("Content-Type", "application/json")

	// Execute HTTP request
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending HTTP request: %v", err)
		return "", err
	}
	defer res.Body.Close()

	// Read response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return "", err
	}

	// Log and return the response from email service
	responseBody := string(body)
	log.Printf("Response from email service: %s", responseBody)

	return responseBody, nil
}
