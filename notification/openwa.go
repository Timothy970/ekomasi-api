package notification

import (
	"bytes"
	"ekomasi_backend/config"
	"ekomasi_backend/dtos"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
}

var (
	defaultClient *Client
	clientOnce    sync.Once
)

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
	}
}

// GetClient returns a global singleton OpenWA client initialized with app configuration
func GetClient() *Client {
	clientOnce.Do(func() {
		cfg := config.Get()
		defaultClient = NewClient(
			cfg.OpenWA.BaseURL,
			cfg.OpenWA.APIKey,
		)
	})
	return defaultClient
}

// GetSessions helps get all active sessions on the server
func GetSessions() error {
	client := GetClient()
	resp, status, err := client.DoRequest(
		http.MethodGet,
		"/api/sessions?limit=100&offset=0",
		nil,
	)

	if err != nil {
		log.Printf("Failed to get sessions: %v", err)
		return err
	}

	fmt.Printf("OpenWA Status: %d\nResponse: %s\n", status, string(resp))
	return nil
}

/*
================================================================================
SAMPLE OPENWA TEMPLATE USAGE: GUEST CHECKOUT ACCOUNT CONVERSION INVITATION
================================================================================
To dispatch an automated WhatsApp template when a guest buyer completes checkout,
use the templateName "guest_account_invite" registered in your OpenWA template manager.

Example Template JSON Payload sent to OpenWA:
{
  "chatId": "254712345678@c.us",
  "templateName": "guest_account_invite",
  "vars": {
    "1": "Alice",                                     // Customer Name
    "2": "ORD-9843",                                  // Order ID
    "3": "https://ekomasi.shop/set-password?token=XYZ" // Account Setup Link
  }
}

Example Code Call:
func SendGuestCheckoutAccountInvite(phone string, customerName string, orderID string, activationToken string) error {
	templateData := map[string]any{
		"1": customerName,
		"2": orderID,
		"3": fmt.Sprintf("%s/set-password?token=%s", config.Get().Server.FrontEndURL, activationToken),
	}
	return SendTemplateMessage(phone, "guest_account_invite", templateData)
}
================================================================================
*/

// SendTemplateMessage helps send a template message via OpenWA API
// Param to: int
// Param templateName: string
// Param templateData: map[string]any
// Return error
func SendTemplateMessage(to string, templateName string, templateData map[string]any) error {
	client := GetClient()

	chatID := to + "@c.us"
	payload := dtos.SendTemplateRequest{
		ChatID:       chatID,
		TemplateName: templateName,
		Vars:         templateData,
	}
	log.Println("Sending template message to:", to, "with payload", payload, "session ", config.Get().OpenWA.Session)
	session := config.Get().OpenWA.Session
	resp, status, err := client.DoRequest(
		http.MethodPost,
		"/api/sessions/"+session+"/messages/send-template",
		payload,
	)

	if err != nil {
		log.Printf("Failed to send template message: %v", err)
		return err
	}

	fmt.Printf("OpenWA Status: %d\nResponse: %s\n", status, string(resp))
	return nil
}

// SendTextMessage helps send a text message via OpenWA API
// Param to: int
// Param message: string
// @return error
//
//	curl -X POST "$BASE/api/sessions/my-session/messages/send-text" \
//	  -H "X-API-Key: $API_KEY" \
//	  -H "Content-Type: application/json" \
//	  -d '{ "chatId": "628123456789@c.us", "text": "Hello from OpenWA!" }'
func SendTextMessage(to string, message string) error {
	client := GetClient()
	chatID := to + "@c.us"
	payload := dtos.SendTextRequest{
		ChatID: chatID,
		Text:   message,
	}
	log.Println("Sending text message to:", to, "with payload", payload, "session ", config.Get().OpenWA.Session)
	session := config.Get().OpenWA.Session
	resp, status, err := client.DoRequest(
		http.MethodPost,
		"/api/sessions/"+session+"/messages/send-text",
		payload,
	)
	if err != nil {
		log.Printf("Failed to send text message: %v", err)
		return err
	}

	fmt.Printf("OpenWA Status: %d\nResponse: %s\n", status, string(resp))
	return nil
}

func (c *Client) DoRequest(method, endpoint string, payload any) ([]byte, int, error) {
	var body io.Reader

	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, body)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)

	// Instantiate HTTP client inside DoRequest
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
