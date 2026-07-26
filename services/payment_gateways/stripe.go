package payment_gateways

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ekomasi_backend/dtos"
)

type StripeService struct {
	SecretKey string
	PublicKey string
}

func NewStripeService(secretKey, publicKey string) *StripeService {
	return &StripeService{
		SecretKey: secretKey,
		PublicKey: publicKey,
	}
}

func (s *StripeService) InitializeTransaction(req dtos.PaymentInitializeRequest) (*dtos.PaymentInitializeResponse, error) {
	apiURL := "https://api.stripe.com/v1/checkout/sessions"

	currency := strings.ToLower(req.Currency)
	if currency == "" {
		currency = "usd" // Default for Stripe if not specified
	}

	reference := req.OrderID + "-" + fmt.Sprintf("%d", time.Now().Unix())
	unitAmount := int64(req.Amount * 100)

	data := url.Values{}
	data.Set("payment_method_types[0]", "card")
	data.Set("line_items[0][price_data][currency]", currency)
	data.Set("line_items[0][price_data][product_data][name]", "Order #"+req.OrderID)
	data.Set("line_items[0][price_data][unit_amount]", fmt.Sprintf("%d", unitAmount))
	data.Set("line_items[0][quantity]", "1")
	data.Set("mode", "payment")
	data.Set("customer_email", req.Email)
	data.Set("client_reference_id", reference)

	successURL := req.CallbackURL
	if successURL == "" {
		successURL = "https://example.com/payment/success?session_id={CHECKOUT_SESSION_ID}"
	}
	cancelURL := req.CallbackURL
	if cancelURL == "" {
		cancelURL = "https://example.com/payment/cancel"
	}
	data.Set("success_url", successURL)
	data.Set("cancel_url", cancelURL)

	httpReq, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create stripe request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.SecretKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stripe HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read stripe response: %w", err)
	}

	var stripeSession struct {
		ID         string `json:"id"`
		URL        string `json:"url"`
		Status     string `json:"status"`
		Error      struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &stripeSession); err != nil {
		return nil, fmt.Errorf("failed to parse stripe session response: %w", err)
	}

	if stripeSession.Error.Message != "" {
		return nil, fmt.Errorf("stripe initialization failed: %s", stripeSession.Error.Message)
	}

	return &dtos.PaymentInitializeResponse{
		AuthorizationURL: stripeSession.URL,
		AccessCode:       stripeSession.ID,
		Reference:        reference,
		GatewayName:      "stripe",
		Status:           true,
		Message:          "Stripe Checkout Session initialized successfully",
	}, nil
}

func (s *StripeService) VerifyTransaction(sessionID string) (*dtos.PaymentVerifyResponse, error) {
	apiURL := fmt.Sprintf("https://api.stripe.com/v1/checkout/sessions/%s", sessionID)

	httpReq, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create stripe verify request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.SecretKey)

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stripe verify HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	var sessionResp struct {
		ID                string  `json:"id"`
		PaymentStatus     string  `json:"payment_status"` // 'paid', 'unpaid'
		AmountTotal       float64 `json:"amount_total"`
		Currency          string  `json:"currency"`
		CustomerEmail     string  `json:"customer_email"`
		ClientReferenceID string  `json:"client_reference_id"`
	}

	if err := json.NewDecoder(res.Body).Decode(&sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode stripe verify response: %w", err)
	}

	statusStr := "FAILED"
	if sessionResp.PaymentStatus == "paid" {
		statusStr = "SUCCESS"
	}

	return &dtos.PaymentVerifyResponse{
		Reference:     sessionResp.ClientReferenceID,
		GatewayName:   "stripe",
		Status:        statusStr,
		Amount:        sessionResp.AmountTotal / 100.0,
		Currency:      strings.ToUpper(sessionResp.Currency),
		PaidAt:        time.Now(),
		Channel:       "card",
		CustomerEmail: sessionResp.CustomerEmail,
	}, nil
}
