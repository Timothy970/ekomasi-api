package payments

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	contentTypeJSON = "application/json"
	content         = "Content-Type"
)

func HandleMpesaPayment(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	var req dtos.MpesaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to decode MPESA payment request",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid request",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	order, err := models.GetOrderByID(req.OrderID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get order for MPESA payment",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	log.Printf("order found %v", order)
	req.Amount = int(order.TotalAmount) - int(order.TotalDiscount)
	req.DeliveryID = order.DeliveryID
	req.Reference = randString(12)
	req.Description = fmt.Sprintf("Payment for order %s", order.OrderID)
	client, err := NewMpesaClient()
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to initialize MPESA client",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("Failed to initialize MPESA client %s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	response, err := client.LipaNaMpesaOnline(req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to initiate MPESA payment",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	err = models.StoreStkResponse(response, req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to store MPESA payment request",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "MPESA payment request initiated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   response,
		Message:   "Payment request initiated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
func RegisterMpesaRoutesHandler(w http.ResponseWriter, r *http.Request) {
	client, err := NewMpesaClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := client.RegisterURLs(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("M-Pesa URLs registered successfully"))
}

type MpesaClient struct {
	ConsumerKey    string
	ConsumerSecret string
	ShortCode      string
	Passkey        string
	CallbackURL    string
	AccessToken    string
	MpesaURL       string
}

func NewMpesaClient() (*MpesaClient, error) {
	client := &MpesaClient{
		ConsumerKey:    os.Getenv("MPESA_CONSUMER_KEY"),
		ConsumerSecret: os.Getenv("MPESA_CONSUMER_SECRET"),
		ShortCode:      os.Getenv("MPESA_SHORTCODE"),
		Passkey:        os.Getenv("MPESA_PASSKEY"),
		CallbackURL:    os.Getenv("MPESA_CALLBACK_URL"),
		MpesaURL:       os.Getenv("MPESA_SEND_URL"),
	}
	err := client.generateToken()
	return client, err
}

func (m *MpesaClient) generateToken() error {
	url := fmt.Sprintf("%soauth/v1/generate?grant_type=client_credentials", m.MpesaURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(m.ConsumerKey, m.ConsumerSecret)
	req.Header.Set(content, contentTypeJSON)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to get token, status: %d, body: %s", res.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return fmt.Errorf("access_token not found in response: %v", result)
	}

	m.AccessToken = token
	return nil
}

func (m *MpesaClient) LipaNaMpesaOnline(paymentRequest dtos.MpesaRequest) (map[string]interface{}, error) {
	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString([]byte(m.ShortCode + m.Passkey + timestamp))

	payload := map[string]interface{}{
		"BusinessShortCode": m.ShortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerPayBillOnline", // or CustomerPayBillOnline
		"Amount":            paymentRequest.Amount,
		"PartyA":            paymentRequest.Phone,
		"PartyB":            m.ShortCode,
		"PhoneNumber":       paymentRequest.Phone,
		"CallBackURL":       m.CallbackURL,
		"AccountReference":  paymentRequest.Reference,
		"TransactionDesc":   paymentRequest.Description,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%smpesa/stkpush/v1/processrequest", m.MpesaURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.AccessToken)
	req.Header.Set(content, contentTypeJSON)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (m *MpesaClient) RegisterURLs() error {
	url := fmt.Sprintf("%smpesa/c2b/v1/registerurl", m.MpesaURL)

	payload := map[string]string{
		"ShortCode":       m.ShortCode,
		"ResponseType":    "Completed",
		"ConfirmationURL": m.CallbackURL,
		"ValidationURL":   m.CallbackURL,
	}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+m.AccessToken)
	req.Header.Set(content, contentTypeJSON)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	log.Println("RegisterURL response:", string(body))
	return nil
}

// Handler for the MPesa callback
func HandleMpesaCallback(w http.ResponseWriter, r *http.Request) {
	//log the IP address of the caller
	log.Printf("MPESA CALLBACK FROM IP: %s", r.RemoteAddr)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	log.Printf("callback body:::::%v", string(bodyBytes))

	var callback dtos.STKCallbackRequest
	if err := json.Unmarshal(bodyBytes, &callback); err != nil {
		http.Error(w, "failed to parse callback", http.StatusBadRequest)
		return
	}

	stk := callback.Body.StkCallback
	logCallbackInfo(stk.CheckoutRequestID, stk.ResultDesc)

	if stk.ResultCode == 0 {
		handleSuccessfulPayment(w, callback)
		return
	}

	handleFailedPayment(w, stk.ResultCode, stk.ResultDesc)
}

// logCallbackInfo logs callback metadata
func logCallbackInfo(checkoutID, resultDesc string) {
	log.Printf("MPESA CALLBACK RECEIVED:\n- CheckoutRequestID: %s\n- Result: %s\n- Time: %s\n",
		checkoutID, resultDesc, time.Now().Format(time.RFC3339))
}

// handleSuccessfulPayment processes successful Mpesa payments
func handleSuccessfulPayment(w http.ResponseWriter, callback dtos.STKCallbackRequest) {
	stk := callback.Body.StkCallback
	amount, mpesaCode, phone := extractMetadata(stk.CallbackMetadata.Item)

	log.Printf("SUCCESSFUL PAYMENT:\n- Phone: %s\n- Amount: %.2f\n- Code: %s\n", phone, amount, mpesaCode)

	deliveryID, orderID, orderType, err := models.UpdateStkResponse(callback, "SUCCESS")
	if err != nil {
		log.Printf("error updating STK response: %v", err)
	}

	processOrderUpdate(orderType, deliveryID, orderID)
	//send sms and email notification
	//store the order to order_notifications table for processing later
	models.StoreOrderNotification(orderID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
}

// handleFailedPayment logs failed payments and responds OK
func handleFailedPayment(w http.ResponseWriter, resultCode int, resultDesc string) {
	log.Printf("FAILED PAYMENT:\n- Code: %d\n- Desc: %s\n", resultCode, resultDesc)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Callback received"}`))
}

// ✅ extractMetadata now matches the exact struct definition in your DTO
func extractMetadata(items []struct {
	Name  string      `json:"Name"`
	Value interface{} `json:"Value"`
}) (float64, string, string) {
	var amount float64
	var mpesaCode, phone string

	for _, item := range items {
		switch item.Name {
		case "Amount":
			if v, ok := item.Value.(float64); ok {
				amount = v
			}
		case "MpesaReceiptNumber":
			if v, ok := item.Value.(string); ok {
				mpesaCode = v
			}
		case "PhoneNumber":
			if v, ok := item.Value.(float64); ok {
				phone = fmt.Sprintf("%.0f", v)
			}
		}
	}
	return amount, mpesaCode, phone
}

func processOrderUpdate(orderType, deliveryID, orderID string) {
	var err error

	switch strings.ToLower(orderType) {
	case "voucher":
		err = models.UpdateVoucherOrderTables(orderID)
		if err != nil {
			log.Printf("error updating voucher order: %v", err)
		}
		utils.SendToUser("", orderID, "voucher_order", buildPaymentSuccessPayload(orderID, nil))
	default:
		err = models.UpdateDeliveryOrderTables(deliveryID, orderID)
		if err != nil {
			log.Printf("error updating delivery order: %v", err)
		}
		utils.SendToUser("", orderID, deliveryID, buildPaymentSuccessPayload(orderID, deliveryID))
	}
}

func buildPaymentSuccessPayload(orderID, deliveryID interface{}) map[string]interface{} {
	return map[string]interface{}{
		"event":       "payment_success",
		"message":     "Your payment was successful!",
		"order_id":    orderID,
		"delivery_id": deliveryID,
	}
}

// randString generates a random alphanumeric string of the given length.
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = letters[i%len(letters)]
		}
		return string(b)
	}
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

func HandleMpesaVoucherPayment(orderID, phoneNumber string, amount float64) error {

	req := &dtos.MpesaRequest{
		OrderID: orderID,
		Phone:   phoneNumber,
		// Amount:      int(amount),
		Amount:      1,
		DeliveryID:  "",
		Reference:   randString(12),
		Description: fmt.Sprintf("Payment for voucher order %s", orderID),
		Type:        "VOUCHER",
	}
	client, err := NewMpesaClient()
	if err != nil {
		return err
	}
	response, err := client.LipaNaMpesaOnline(*req)
	if err != nil {
		return err
	}
	err = models.StoreStkResponse(response, *req)
	if err != nil {
		return err
	}
	return nil
}
