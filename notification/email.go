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

// SendEmail sends an email to the specified recipient with the given subject and body.
func SendEmail(to string, subject string, body string) error {
	newPayload := map[string]interface{}{
		"apikey":       os.Getenv("APIKEY"),
		"partnerID":    os.Getenv("PARTNERID"),
		"from_address": os.Getenv("SENDER_EMAIL"),
		"to_address":   to,
		"body":         body,
		"subject":      subject,
		"time":         time.Now(),
		"date":         time.Now().Format("2006-01-02"),
		"scheduled":    false,
		"attachments":  []interface{}{},
	}
	// // Send the email and get the response
	responseBody, err := sendSingleEmail(newPayload)
	if err != nil {
		log.Printf("Error sending email: %v", err)
	}
	log.Printf("Email sent to %s with subject '%s'. Response: %s", to, subject, responseBody)
	return err
}

// function to send single emails to apiv2
func sendSingleEmail(data map[string]interface{}) (string, error) {
	sendUrl := os.Getenv("V2_URL")
	url := sendUrl + "/services/send-email"
	method := "POST"
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return "", err
	}

	payload := strings.NewReader(string(payloadBytes))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending HTTP request: %v", err)
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return "", err
	}

	// Log the response from the email service
	responseBody := string(body)
	log.Printf("Response from email service: %s", responseBody)

	// Return the response body and nil error
	return responseBody, nil
}
