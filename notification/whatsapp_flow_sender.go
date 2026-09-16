package notification

import (
	"bytes"
	"ekomasi_backend/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type FlowInteractivePayload struct {
	MessagingProduct string         `json:"messaging_product"`
	RecipientType    string         `json:"recipient_type"`
	To               string         `json:"to"`
	Type             string         `json:"type"` // "interactive"
	Interactive      map[string]any `json:"interactive"`
}

// SendWhatsAppFlowMessage sends an interactive WhatsApp Flow invitation message to a recipient
func SendWhatsAppFlowMessage(toPhone string, flowID string, flowToken string, ctaText string, initialScreen string) error {
	cfg := config.Get()
	if cfg.WhatsAppCloud.PhoneNumberID == "" || cfg.WhatsAppCloud.AccessToken == "" {
		log.Println("[WhatsApp Flow Sender] Skipping send: WHATSAPP_PHONE_NUMBER_ID or WHATSAPP_ACCESS_TOKEN not configured")
		return nil
	}

	url := fmt.Sprintf("https://graph.facebook.com/v20.0/%s/messages", cfg.WhatsAppCloud.PhoneNumberID)

	payload := FlowInteractivePayload{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               toPhone,
		Type:             "interactive",
		Interactive: map[string]any{
			"type": "flow",
			"header": map[string]any{
				"type": "text",
				"text": "Ekomasi Interactive Store",
			},
			"body": map[string]any{
				"text": "Tap below to complete your action directly inside WhatsApp.",
			},
			"footer": map[string]any{
				"text": "Powered by Ekomasi",
			},
			"action": map[string]any{
				"name": "flow",
				"parameters": map[string]any{
					"flow_message_version": "3.0",
					"flow_token":           flowToken,
					"flow_id":              flowID,
					"flow_cta":             ctaText,
					"flow_action":          "navigate",
					"flow_action_payload": map[string]any{
						"screen": initialScreen,
					},
					"mode": "draft",
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal flow message: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("create flow request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.WhatsAppCloud.AccessToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send flow request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Meta API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[WhatsApp Flow Sender] Sent flow %s to %s successfully", flowID, toPhone)
	return nil
}
