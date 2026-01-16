// Package notification provides SMS notification services for the Adenzo e-commerce platform.
//
// This file handles:
//   - SMS sending via external API (V2 SMS service)
//   - SMS payload construction with authentication
//   - Mobile number validation and message delivery
//   - HTTP client communication with SMS gateway
//   - Shortcode-based SMS sending
//
// Environment variables required:
//   - SMSAPIKEY: API key for SMS service authentication
//   - SMSPARTNERID: Partner ID for SMS service
//   - SHORTCODE: SMS shortcode/sender ID
//   - V2_URL: Base URL for the V2 SMS service API
package notification

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// SendSmsMessages sends an SMS message to the specified mobile number.
//
// This function uses the Emalify SMS API (V2 service) to send text messages.
// It constructs the SMS payload with authentication credentials from environment
// variables and sends it via HTTP POST request.
//
// Parameters:
//   - to: string - Recipient mobile number (should include country code)
//   - message: string - SMS message content (plain text)
//
// Returns:
//   - None (void function)
func SendSmsMessages(to, message string) {
	// Construct SMS API endpoint URL
	url := os.Getenv("V2_URL")
	url = strings.TrimSuffix(url, "/") + "/services/sendsms" // Ensure no double slashes
	method := "POST"

	// Construct SMS payload with authentication and message data
	newPayload := map[string]interface{}{
		"apikey":    os.Getenv("SMSAPIKEY"),    // SMS API authentication key
		"partnerID": os.Getenv("SMSPARTNERID"), // Partner identification
		"mobile":    to,                        // Recipient mobile number
		"message":   message,                   // SMS text content
		"shortcode": os.Getenv("SHORTCODE"),    // Sender ID/shortcode
		"pass_type": "plain",                   // Plain text message type
	}

	// Marshal payload to JSON format
	payloadBytes, err := json.Marshal(newPayload)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return
	}

	// Create JSON reader for request body
	payload := strings.NewReader(string(payloadBytes))

	// Initialize HTTP client
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return
	}

	// Set content type header for JSON payload
	req.Header.Add("Content-Type", "application/json")

	// Execute HTTP request to SMS gateway
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending HTTP request: %v", err)
		return
	}
	defer res.Body.Close()

	// Read response body from SMS service
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return
	}

	// Log successful SMS delivery with response
	log.Printf("SMS sent to %s with response: %s", to, string(body))
}
