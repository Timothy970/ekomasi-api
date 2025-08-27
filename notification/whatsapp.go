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

func SendWhatsappMessages(to int, message string, template string) error {

	url := os.Getenv("WHATSAPPSENDURL")
	method := "POST"

	newPayload, err := createSendPayload(template, message, to)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return err
	}
	payloadBytes, err := json.Marshal(newPayload)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return err
	}
	payload := strings.NewReader(string(payloadBytes))
	client := &http.Client{}

	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending HTTP request: %v", err)

		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)

		return err
	}
	responseBody := string(body)
	log.Printf("WhatsApp OTP sent to %d with response: %s", to, responseBody)
	return nil
}

// helper function to create payloads for whatsapp
func createSendPayload(template, message string, to int) (map[string]interface{}, error) {
	// Validate and parse environment variables
	partnerID, err := strconv.Atoi(os.Getenv("PARTNERID"))
	if err != nil {
		return nil, fmt.Errorf("invalid partner id: %v", err)
	}

	sender, err := strconv.Atoi(os.Getenv("WHATSAPPSENDER"))
	if err != nil {
		return nil, fmt.Errorf("invalid sender: %v", err)
	}

	// Create payload based on template type
	switch template {
	case "Otp":
		return map[string]interface{}{
			"partner_api_key": os.Getenv("APIKEY"),
			"partner_id":      partnerID,
			"template_name":   "auth_otp_template",
			"sender":          sender,
			"category":        "authentication",
			"recipient":       to,
			"otp":             message,
		}, nil

	case "Auth":
		return map[string]interface{}{
			"partner_id":      partnerID,
			"partner_api_key": os.Getenv("APIKEY"),
			"sender":          sender,
			"template_name":   "user_verification",
			"category":        "UTILITY",
			"recipients": []map[string]interface{}{
				{
					"phone_number":        to,
					"buttonURL_variables": []string{message},
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown template type: %s", template)
	}
}
