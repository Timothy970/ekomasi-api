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

type PaystackService struct {
	SecretKey string
	PublicKey string
}

func NewPaystackService(secretKey, publicKey string) *PaystackService {
	return &PaystackService{
		SecretKey: secretKey,
		PublicKey: publicKey,
	}
}

type paystackInitRequest struct {
	Email       string `json:"email"`
	Amount      int64  `json:"amount"` // in kobo/cents (amount * 100)
	Reference   string `json:"reference"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type paystackInitResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

func (s *PaystackService) InitializeTransaction(req dtos.PaymentInitializeRequest) (*dtos.PaymentInitializeResponse, error) {
	url := "https://api.paystack.co/transaction/initialize"

	payload := paystackInitRequest{
		Email:       req.Email,
		Amount:      int64(req.Amount * 100),
		Reference:   req.OrderID + "-" + fmt.Sprintf("%d", time.Now().Unix()),
		CallbackURL: req.CallbackURL,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal paystack payload: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create paystack request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.SecretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("paystack HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read paystack response: %w", err)
	}

	var resp paystackInitResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse paystack response: %w", err)
	}

	if !resp.Status {
		return nil, fmt.Errorf("paystack initialization failed: %s", resp.Message)
	}

	return &dtos.PaymentInitializeResponse{
		AuthorizationURL: resp.Data.AuthorizationURL,
		AccessCode:       resp.Data.AccessCode,
		Reference:        resp.Data.Reference,
		GatewayName:      "paystack",
		Status:           true,
		Message:          resp.Message,
	}, nil
}

func (s *PaystackService) VerifyTransaction(reference string) (*dtos.PaymentVerifyResponse, error) {
	url := fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create verify request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.SecretKey)

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("paystack verify HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	var verifyResp struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Status    string    `json:"status"` // 'success', 'failed'
			Reference string    `json:"reference"`
			Amount    float64   `json:"amount"` // in kobo/cents
			Currency  string    `json:"currency"`
			Channel   string    `json:"channel"`
			PaidAt    time.Time `json:"paid_at"`
			Customer  struct {
				Email string `json:"email"`
			} `json:"customer"`
		} `json:"data"`
	}

	if err := json.NewDecoder(res.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode verify response: %w", err)
	}

	statusStr := "FAILED"
	if verifyResp.Status && verifyResp.Data.Status == "success" {
		statusStr = "SUCCESS"
	}

	return &dtos.PaymentVerifyResponse{
		Reference:     verifyResp.Data.Reference,
		GatewayName:   "paystack",
		Status:        statusStr,
		Amount:        verifyResp.Data.Amount / 100.0,
		Currency:      verifyResp.Data.Currency,
		PaidAt:        verifyResp.Data.PaidAt,
		Channel:       verifyResp.Data.Channel,
		CustomerEmail: verifyResp.Data.Customer.Email,
	}, nil
}
