package payment_gateways

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ekomasi_backend/dtos"
)

type FlutterwaveService struct {
	SecretKey string
	PublicKey string
}

func NewFlutterwaveService(secretKey, publicKey string) *FlutterwaveService {
	return &FlutterwaveService{
		SecretKey: secretKey,
		PublicKey: publicKey,
	}
}

type flwInitRequest struct {
	TxRef         string `json:"tx_ref"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	RedirectURL   string `json:"redirect_url"`
	Customer      flwCustomer `json:"customer"`
	Customizations struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"customizations"`
}

type flwCustomer struct {
	Email string `json:"email"`
}

type flwInitResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Link string `json:"link"`
	} `json:"data"`
}

func (s *FlutterwaveService) InitializeTransaction(req dtos.PaymentInitializeRequest) (*dtos.PaymentInitializeResponse, error) {
	url := "https://api.flutterwave.com/v3/payments"

	txRef := req.OrderID + "-" + fmt.Sprintf("%d", time.Now().Unix())
	currency := req.Currency
	if currency == "" {
		currency = "KES"
	}

	payload := flwInitRequest{
		TxRef:       txRef,
		Amount:      fmt.Sprintf("%.2f", req.Amount),
		Currency:    currency,
		RedirectURL: req.CallbackURL,
		Customer: flwCustomer{
			Email: req.Email,
		},
	}
	payload.Customizations.Title = "Payment for Order " + req.OrderID
	payload.Customizations.Description = "E-Commerce Checkout"

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal flutterwave payload: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create flutterwave request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.SecretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("flutterwave HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read flutterwave response: %w", err)
	}

	var resp flwInitResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse flutterwave response: %w", err)
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("flutterwave initialization failed: %s", resp.Message)
	}

	return &dtos.PaymentInitializeResponse{
		AuthorizationURL: resp.Data.Link,
		Reference:        txRef,
		GatewayName:      "flutterwave",
		Status:           true,
		Message:          resp.Message,
	}, nil
}

func (s *FlutterwaveService) VerifyTransaction(transactionID string) (*dtos.PaymentVerifyResponse, error) {
	url := fmt.Sprintf("https://api.flutterwave.com/v3/transactions/%s/verify", transactionID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create flutterwave verify request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.SecretKey)

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("flutterwave verify HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	var verifyResp struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			ID          int64     `json:"id"`
			TxRef       string    `json:"tx_ref"`
			Amount      float64   `json:"amount"`
			Currency    string    `json:"currency"`
			Status      string    `json:"status"` // 'successful', 'failed'
			CreatedAt   time.Time `json:"created_at"`
			PaymentType string    `json:"payment_type"`
			Customer    struct {
				Email string `json:"email"`
			} `json:"customer"`
		} `json:"data"`
	}

	if err := json.NewDecoder(res.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode flutterwave verify response: %w", err)
	}

	statusStr := "FAILED"
	if verifyResp.Status == "success" && verifyResp.Data.Status == "successful" {
		statusStr = "SUCCESS"
	}

	return &dtos.PaymentVerifyResponse{
		Reference:     verifyResp.Data.TxRef,
		GatewayName:   "flutterwave",
		Status:        statusStr,
		Amount:        verifyResp.Data.Amount,
		Currency:      verifyResp.Data.Currency,
		PaidAt:        verifyResp.Data.CreatedAt,
		Channel:       verifyResp.Data.PaymentType,
		CustomerEmail: verifyResp.Data.Customer.Email,
	}, nil
}
