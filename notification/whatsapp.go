// Package notification provides WhatsApp notification services for the Ekomasi e-commerce platform.
//
// This file handles:
//   - WhatsApp message sending via external API
//   - Template-based messaging (OTP, authentication)
//   - WhatsApp payload construction with authentication
//   - HTTP client communication with WhatsApp gateway
//   - Multiple template support (OTP verification, user authentication)
//
// Environment variables required:
//   - WHATSAPPSENDURL: WhatsApp API endpoint URL
//   - APIKEY: Partner API key for authentication
//   - PARTNERID: Partner ID (must be valid integer)
//   - WHATSAPPSENDER: WhatsApp sender ID (must be valid integer)
//
// Supported templates:
//   - "Otp": One-time password delivery (auth_otp_template)
//   - "Auth": User verification with button URL (user_verification template)
package notification

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// SendWhatsappMessages sends a WhatsApp message using a specific template.
//
// This function sends template-based WhatsApp messages for authentication
// purposes (OTP delivery or user verification). It constructs the appropriate
// payload based on the template type and sends it to the WhatsApp API.
//
// Parameters:
//   - to: int - Recipient phone number (should include country code as integer)
//   - message: string - Message content (OTP code or verification URL parameter)
//   - template: string - Template type ("Otp" or "Auth")
//
// Returns:
//   - error: Payload creation error, marshaling error, HTTP error, or nil on success
func SendWhatsappMessages(to int, message string, template string) error {

	// Get WhatsApp API endpoint from environment
	url := os.Getenv("WHATSAPPSENDURL")
	method := "POST"

	// Create template-specific payload with authentication credentials
	newPayload, err := createSendPayload(template, message, to)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return err
	}

	// Marshal payload to JSON format
	payloadBytes, err := json.Marshal(newPayload)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return err
	}

	// Create JSON reader for request body
	payload := strings.NewReader(string(payloadBytes))
	client := &http.Client{}

	// Create HTTP POST request
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return err
	}

	// Set content type header for JSON payload
	req.Header.Add("Content-Type", "application/json")

	// Execute HTTP request to WhatsApp gateway
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending HTTP request: %v", err)

		return err
	}
	defer res.Body.Close()

	// Read response body from WhatsApp service
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)

		return err
	}

	// Log successful WhatsApp delivery with response
	responseBody := string(body)
	log.Printf("WhatsApp OTP sent to %d with response: %s", to, responseBody)
	return nil
}

// createSendPayload creates template-specific WhatsApp message payloads.
//
// This helper function constructs the appropriate payload structure based on
// the template type. Different templates have different payload structures
// and required fields.
//
// Parameters:
//   - template: string - Template type ("Otp" or "Auth")
//   - message: string - Message content (OTP code or URL parameter)
//   - to: int - Recipient phone number
//
// Returns:
//   - map[string]any: Template-specific payload
//   - error: Environment variable parsing error, unknown template error, or nil on success
func createSendPayload(template, message string, to int) (map[string]any, error) {
	// Parse and validate partner ID from environment
	partnerID, err := strconv.Atoi(os.Getenv("PARTNERID"))
	if err != nil {
		return nil, fmt.Errorf("invalid partner id: %v", err)
	}

	// Parse and validate WhatsApp sender ID from environment
	sender, err := strconv.Atoi(os.Getenv("WHATSAPPSENDER"))
	if err != nil {
		return nil, fmt.Errorf("invalid sender: %v", err)
	}

	// Create template-specific payload
	switch template {
	case "Otp":
		// OTP template: Single recipient with OTP code
		return map[string]any{
			"partner_api_key": os.Getenv("APIKEY"), // API authentication key
			"partner_id":      partnerID,           // Partner identification
			"template_name":   "auth_otp_template", // OTP template name
			"sender":          sender,              // WhatsApp sender ID
			"category":        "authentication",    // Message category
			"recipient":       to,                  // Recipient phone number
			"otp":             message,             // OTP code to send
		}, nil

	case "Auth":
		// Auth template: User verification with button URL parameter
		return map[string]any{
			"partner_id":      partnerID,           // Partner identification
			"partner_api_key": os.Getenv("APIKEY"), // API authentication key
			"sender":          sender,              // WhatsApp sender ID
			"template_name":   "user_verification", // Verification template name
			"category":        "UTILITY",           // Message category
			"recipients": []map[string]any{ // Array of recipients
				{
					"phone_number":        to,                // Recipient phone number
					"buttonURL_variables": []string{message}, // URL parameter for button
				},
			},
		}, nil

	default:
		// Return error for unsupported template types
		return nil, fmt.Errorf("unknown template type: %s", template)
	}
}
