package notification

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func SendSmsMessages(to, message string) {
	// This function sends an SMS message using the Emalify API.
	url := os.Getenv("V2_URL")
	url = strings.TrimSuffix(url, "/") + "/services/sendsms"
	method := "POST"

	newPayload := map[string]interface{}{
		"apikey":    os.Getenv("SMSAPIKEY"),
		"partnerID": os.Getenv("SMSPARTNERID"),
		"mobile":    to,
		"message":   message,
		"shortcode": os.Getenv("SHORTCODE"),
		"pass_type": "plain",
	}
	payloadBytes, err := json.Marshal(newPayload)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return
	}
	payload := strings.NewReader(string(payloadBytes))
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending HTTP request: %v", err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return
	}
	log.Printf("SMS sent to %s with response: %s", to, string(body))
}
