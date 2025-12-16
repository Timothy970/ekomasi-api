package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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
	req, ok := DecodeRequestBody[dtos.MpesaRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
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
	req.Reference = "ADENZO - " + order.OrderID
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

	response, err := client.LipaNaMpesaOnline(*req)
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
	err = models.StoreStkResponse(response, *req)
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
	err = storeTransactionLog(*req)
	if err != nil {
		log.Printf("Failed to store transaction log: %v", err)
	}
	environment := os.Getenv("ENVIRONMENT")

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "MPESA payment request initiated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Payment request initiated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
	if environment == "development" {
		go sendCallbackToDevEnv(req.OrderID)
	}
}

func storeTransactionLog(req dtos.MpesaRequest) error {
	logEntry := &dtos.TransactionsList{
		OrderID:              req.OrderID,
		TransactionReference: req.Reference,
		PhoneNumber:          &req.Phone,
		Amount:               float64(req.Amount),
		Status:               "PENDING",
		PaymentMethod:        "MPESA",
	}
	return models.InsertTransaction(logEntry)
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
	ConsumerKey       string
	ConsumerSecret    string
	ShortCode         string
	Passkey           string
	CallbackURL       string
	BalanceURL        string
	AccessToken       string
	MpesaURL          string
	InitiatorName     string
	InitiatorPassword string
	// CertificatePath    string
	// SecurityCredential string
	ReturnURL  string
	HeadOffice string
}

func NewMpesaClient() (*MpesaClient, error) {
	// CertificatePath := os.Getenv("MPESA_CERTIFICATE_PATH")
	InitiatorPassword := os.Getenv("MPESA_INITIATOR_PASSWORD")
	// pubKey, err := loadPublicKey(CertificatePath)
	// if err != nil {
	// 	log.Fatal("Failed to load public key:", err)
	// }

	// securityCredential, err := generateSecurityCredential(pubKey, InitiatorPassword)
	// if err != nil {
	// 	log.Fatal("Failed to generate SecurityCredential:", err)
	// }
	client := &MpesaClient{
		ConsumerKey:       os.Getenv("MPESA_CONSUMER_KEY"),
		ConsumerSecret:    os.Getenv("MPESA_CONSUMER_SECRET"),
		ShortCode:         os.Getenv("MPESA_SHORTCODE"),
		Passkey:           os.Getenv("MPESA_PASSKEY"),
		CallbackURL:       os.Getenv("MPESA_CALLBACK_URL"),
		BalanceURL:        os.Getenv("MPESA_BALANCE_URL"),
		MpesaURL:          os.Getenv("MPESA_SEND_URL"),
		InitiatorName:     os.Getenv("MPESA_INITIATOR_NAME"),
		InitiatorPassword: InitiatorPassword,
		// SecurityCredential: securityCredential,
		ReturnURL:  os.Getenv("MPESA_RETURN_URL"),
		HeadOffice: os.Getenv("MPESA_HEAD_OFFICE"),
	}
	log.Printf("Mpesa Client Config: %+v", client)
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
	password := base64.StdEncoding.EncodeToString([]byte(m.HeadOffice + m.Passkey + timestamp))

	payload := map[string]interface{}{
		"BusinessShortCode": m.HeadOffice,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerBuyGoodsOnline",
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
		"ShortCode":       m.HeadOffice,
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

	handleFailedPayment(w, callback)

}

// function to send callback to development environment
func sendCallbackToDevEnv(orderID string) {
	//delay for 20 seconds to simulate real payment delay
	time.Sleep(20 * time.Second)
	// using order_id i can get the checkout_request_id

	//hardcoded callback response
	resultCode := 0
	resultDesc := "The service request is processed successfully."
	merchantRequestID := "12345-" + orderID
	checkoutRequestID, err := models.GetCheckoutRequestIDByOrderID(orderID)
	if err != nil {
		resultCode = 1
		resultDesc = "The service request cancelled by user"
	}

	mpesaReceiptNumber := "ABC123XYZ-" + orderID
	transactionDate := time.Now().Format("20060102150405")
	phoneNumber := 254700000000

	req := dtos.STKCallbackRequest{}

	req.Body.StkCallback.MerchantRequestID = merchantRequestID
	req.Body.StkCallback.CheckoutRequestID = checkoutRequestID
	req.Body.StkCallback.ResultCode = resultCode
	req.Body.StkCallback.ResultDesc = resultDesc

	// Build items
	req.Body.StkCallback.CallbackMetadata.Item = []struct {
		Name  string      `json:"Name"`
		Value interface{} `json:"Value"`
	}{
		{
			Name:  "Amount",
			Value: 1,
		},
		{
			Name:  "MpesaReceiptNumber",
			Value: mpesaReceiptNumber,
		},
		{
			Name:  "Balance",
			Value: nil,
		},
		{
			Name: "TransactionDate",
			// Must convert to int because callback uses numeric timestamp
			Value: toInt(transactionDate),
		},
		{
			Name:  "PhoneNumber",
			Value: phoneNumber,
		},
	}
	logCallbackInfo(req.Body.StkCallback.CheckoutRequestID, req.Body.StkCallback.ResultDesc)
	if resultCode == 0 {
		handleSuccessfulPayment(nil, req)
	} else {
		handleFailedPayment(nil, req)
	}
}

func toInt(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// store mpesa receipt number
func storeMpesaMpesaReceiptNumber(items []struct {
	Name  string      `json:"Name"`
	Value interface{} `json:"Value"`
}, orderID string) error {
	var mpesaCode string
	for _, item := range items {
		if item.Name == "MpesaReceiptNumber" {
			if v, ok := item.Value.(string); ok {
				mpesaCode = v
			}
		}
	}
	return models.UpdateMpesaReceiptNumber(mpesaCode, orderID)
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

	deliveryID, orderID, orderType, err := models.UpdateStkResponse(callback.Body.StkCallback.CheckoutRequestID, "SUCCESS")
	if err != nil {
		log.Printf("error updating STK response: %v", err)
	}
	if orderID == "" && deliveryID == "" {
		log.Printf("orderID and deliveryID are both empty, trying sending the callback to development environment")
	}
	processOrderUpdate(orderType, deliveryID, orderID, "COMPLETED")
	//update transaction log
	err = models.UpdateTransactionStatus(orderID, "COMPLETED")
	if err != nil {
		log.Printf("error updating transaction status: %v", err)
	}
	items := stk.CallbackMetadata.Item
	err = storeMpesaMpesaReceiptNumber(items, orderID)
	if err != nil {
		log.Printf("error storing mpesa receipt number: %v", err)
	}

	//send sms and email notification
	//store the order to order_notifications table for processing later
	models.StoreOrderNotification(orderID)
	if w == nil {
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
}

// handleFailedPayment logs failed payments and responds OK
func handleFailedPayment(w http.ResponseWriter, callback dtos.STKCallbackRequest) {
	resultCode := callback.Body.StkCallback.ResultCode
	resultDesc := callback.Body.StkCallback.ResultDesc
	checkoutRequestID := callback.Body.StkCallback.CheckoutRequestID
	deliveryID, orderID, orderType, err := models.UpdateStkResponse(checkoutRequestID, "FAILED")
	if err != nil {
		log.Printf("error updating STK response: %v", err)
	}
	if orderID == "" && deliveryID == "" {
		log.Printf("orderID and deliveryID are both empty, trying sending the callback to development environment")
	}
	processOrderUpdate(orderType, deliveryID, orderID, "FAILED")
	//update transaction log
	err = models.UpdateTransactionStatus(orderID, "FAILED")
	if err != nil {
		log.Printf("error updating transaction status: %v", err)
	}
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

func processOrderUpdate(orderType, deliveryID, orderID, status string) {
	var err error

	switch strings.ToLower(orderType) {
	case "voucher":
		err = models.UpdateVoucherOrderTables(orderID, status)
		if err != nil {
			log.Printf("error updating voucher order: %v", err)
		}
		utils.SendToUser("", orderID, "voucher_order", buildPaymentSuccessPayload(orderID, nil, status))
	default:
		err = models.UpdateDeliveryOrderTables(deliveryID, orderID, status)
		if err != nil {
			log.Printf("error updating delivery order: %v", err)
		}
		utils.SendToUser("", orderID, deliveryID, buildPaymentSuccessPayload(orderID, deliveryID, status))
	}
}

func buildPaymentSuccessPayload(orderID, deliveryID interface{}, status string) map[string]interface{} {
	event := "payment_failed"
	if status == "COMPLETED" {
		event = "payment_success"
	}
	message := "There was an issue with your payment."
	if status == "COMPLETED" {
		message = "Your payment was successful!"
	}
	return map[string]interface{}{
		"event":       event,
		"message":     message,
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
		OrderID:     orderID,
		Phone:       phoneNumber,
		Amount:      int(amount),
		DeliveryID:  "",
		Reference:   "ADENZO VOUCHER -" + orderID,
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
func loadPublicKey(certPath string) (*rsa.PublicKey, error) {
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not RSA public key")
	}
	return pubKey, nil
}
func generateSecurityCredential(pubKey *rsa.PublicKey, initiatorPassword string) (string, error) {
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(initiatorPassword))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}
func (m *MpesaClient) FetchPayBillBalance() (map[string]any, error) {

	payload := map[string]interface{}{
		"Initiator": m.InitiatorName,
		// "SecurityCredential": m.SecurityCredential,
		"CommandID":       "AccountBalance",
		"PartyA":          m.ShortCode,
		"IdentifierType":  "4",
		"Remarks":         "balance",
		"QueueTimeOutURL": m.BalanceURL,
		"ResultURL":       m.BalanceURL,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error json payload %s", err)
		return map[string]any{}, err
	}

	url := fmt.Sprintf("%s/mpesa/accountbalance/v1/query", m.MpesaURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request %s", err)
		return map[string]any{}, err
	}
	req.Header.Set("Authorization", "Bearer "+m.AccessToken)
	req.Header.Set(content, contentTypeJSON)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request %s", err)
		return map[string]any{}, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding response %s", err)
		return map[string]any{}, err
	}
	return result, nil

}

// Handler for the MPesa callback
func HandleMpesaBalance(w http.ResponseWriter, r *http.Request) {
	log.Printf("MPESA CALLBACK FROM IP: %s", r.RemoteAddr)

	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	client, err := NewMpesaClient()
	if err != nil {
		http.Error(w, "failed to create mpesa client", http.StatusInternalServerError)
		return
	}
	result, err := client.FetchPayBillBalance()
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{

				Module:      "Payments",
				Description: "Failed to fetch MPESA PayBill balance",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("Failed to fetch MPESA PayBill balance %s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	resultJSON, _ := json.Marshal(result)
	log.Printf("callback body:::::%v", string(resultJSON))
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "MPESA PayBill balance fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   result,
		Message:   "PayBill balance fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func HandleMpesaBalanceCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("MPESA BALANCE CALLBACK FROM IP: %s", r.RemoteAddr)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	log.Printf("balance callback body:::::%v", string(bodyBytes))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
}

type MpesaMoneyReturnRequest struct {
	OrderID     string `json:"order_id" validate:"required"`
	PhoneNumber string `json:"phone_number" validate:"required"`
}

func HandleMpesaReturnCallback(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	log.Printf("balance callback body:::::%v", string(bodyBytes))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
}

func HandleMpesaMoneyReturn(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Accounts")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[MpesaMoneyReturnRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Accounts") {
		return
	}
	order, err := models.GetOrderByID(req.OrderID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Order not found for MPESA money return " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
		})
		return
	}
	client, err := NewMpesaClient()
	if err != nil {
		http.Error(w, "failed to create mpesa client", http.StatusInternalServerError)
		return
	}

	result, err := client.HandleMoneyReturn(order.TotalAmount, req.PhoneNumber)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to handle MPESA money return",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("Failed to handle MPESA money return %s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	if result.ResultCode != "0" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "MPESA money return failed",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("MPESA money return failed: %s", result.ResultDesc),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	//mark the order as refunded and restock items
	err = models.HandleMpesaMoneyReturnRefunds(*order)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to process refunds for MPESA money return",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("Failed to process refunds for MPESA money return %s", err),
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
			Description: "MPESA money return handled successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "MPESA money return handled successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func (m *MpesaClient) HandleMoneyReturn(amount float64, phoneNumber string) (MpesaMoneyReturnResponse, error) {
	payload := map[string]interface{}{
		"OriginatorConversationID": randString(24),
		"InitiatorName":            m.InitiatorName,
		// "SecurityCredential":       m.SecurityCredential,
		"CommandID":       "BusinessPayment",
		"Amount":          amount,
		"PartyA":          m.ShortCode,
		"PartyB":          phoneNumber,
		"Remarks":         "Payment Return for order to phone number " + phoneNumber,
		"QueueTimeOutURL": m.ReturnURL,
		"ResultURL":       m.ReturnURL,
		"Occasion":        "",
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error json payload %s", err)
		return MpesaMoneyReturnResponse{}, err
	}
	url := fmt.Sprintf("%s/mpesa/b2c/v3/paymentrequest", m.MpesaURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request %s", err)
		return MpesaMoneyReturnResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+m.AccessToken)
	req.Header.Set(content, contentTypeJSON)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request %s", err)
		return MpesaMoneyReturnResponse{}, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding response %s", err)
		return MpesaMoneyReturnResponse{}, err
	}
	response := MpesaMoneyReturnResponse{
		ResultCode:               fmt.Sprintf("%v", result["ResponseCode"]),
		ResultDesc:               fmt.Sprintf("%v", result["ResponseDescription"]),
		ConversationID:           fmt.Sprintf("%v", result["ConversationID"]),
		OriginatorConversationID: fmt.Sprintf("%v", result["OriginatorConversationID"]),
	}
	return response, nil
}

type MpesaMoneyReturnResponse struct {
	ResultCode               string `json:"ResultCode"`
	ResultDesc               string `json:"ResultDesc"`
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
}

func HandleMpesaTransactionStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.MpesaTransactionStatus](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}
	// Save a "pending" mark in Redis (expires in 2 minutes)
	pending := map[string]interface{}{
		"status": "pending",
	}
	utils.SetCache("mpesa_status:"+req.TransactionID, pending, 2*time.Minute)
	client, err := NewMpesaClient()
	if err != nil {
		utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to initialize MPESA client",
				Code:        http.StatusOK,
			},
			Payload: dtos.TransactionStatusResponse{
				Status:                   "Failed",
				TransactionID:            req.TransactionID,
				Message:                  "Failed to get transaction status!",
				MpesaReference:           "",
				ConversationID:           "",
				OriginatorConversationID: "",
			},
			Message:   fmt.Sprintf("Failed to initialize MPESA client %s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	response, err := client.CheckMpesaTransactionStatus(*req)
	if err != nil {
		utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to initiate MPESA payment",
				Code:        http.StatusOK,
			},
			Payload: dtos.TransactionStatusResponse{
				Status:                   "Failed",
				TransactionID:            req.TransactionID,
				Message:                  "Failed to get transaction status!",
				MpesaReference:           "",
				ConversationID:           "",
				OriginatorConversationID: "",
			},
			Message:   "Failed to get transaction status!",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	if response.ResponseCode != "0" {
		log.Printf("MPESA transaction status fetch failed: %s", response.ResponseDescription)
		utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "MPESA transaction status fetch failed",
				Code:        http.StatusOK,
			},
			Payload: dtos.TransactionStatusResponse{
				Status:                   "Failed",
				TransactionID:            req.TransactionID,
				Message:                  "MPESA transaction status fetch failed",
				MpesaReference:           "",
				ConversationID:           response.ConversationID,
				OriginatorConversationID: response.OriginatorConversationID,
			},
			Message:   "Failed to get transaction status!",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Wait for callback (poll Redis)
	var callback dtos.MpesaResultResponse
	maxWait := 30 * time.Second
	interval := 1 * time.Second
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		statusObj := struct {
			Status string                   `json:"status"`
			Data   dtos.MpesaResultResponse `json:"data"`
		}{}

		err := utils.GetCache("mpesa_status:"+req.TransactionID, &statusObj)
		if err == nil && statusObj.Status == "done" {
			// Got callback!
			callback = statusObj.Data
			break
		}

		time.Sleep(interval)
	}

	if callback.Result.ResultCode != 0 {
		log.Printf("MPESA transaction status fetch failed: %s", callback.Result.ResultDesc)
		utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "MPESA transaction status fetch failed",
				Code:        http.StatusOK,
			},
			Payload: dtos.TransactionStatusResponse{
				Status:                   "Failed",
				TransactionID:            req.TransactionID,
				Message:                  "MPESA transaction status fetch failed",
				MpesaReference:           "",
				ConversationID:           response.ConversationID,
				OriginatorConversationID: response.OriginatorConversationID,
			},
			Message:   "Failed to get transaction status!",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// Delete from Redis after serving
	utils.DeleteCache("mpesa_status:" + req.TransactionID)

	log.Printf("MPESA transaction status response: %+v", response)
	mpesaReference := GetResultParameterValue(callback.Result.ResultParameters.ResultParameter, "ReceiptNo")

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "MPESA transaction status fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: dtos.TransactionStatusResponse{
			Status:                   "Failed",
			TransactionID:            req.TransactionID,
			Message:                  "MPESA transaction status fetch failed",
			MpesaReference:           mpesaReference,
			ConversationID:           callback.Result.ConversationID,
			OriginatorConversationID: callback.Result.OriginatorConversationID,
		},
		Message:   "Transaction status fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})

}

func GetResultParameterValue(params []dtos.ResultParameter, key string) string {
	for _, param := range params {
		if param.Key == key {
			return param.Value
		}
	}
	return ""
}

func (m *MpesaClient) CheckMpesaTransactionStatus(req dtos.MpesaTransactionStatus) (*dtos.MpesaTransactionStatusRequest, error) {
	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString([]byte(m.HeadOffice + m.Passkey + timestamp))
	payload := map[string]interface{}{
		"Initiator":          "Adenzo",
		"SecurityCredential": password,
		"CommandID":          "TransactionStatusQuery",
		"TransactionID":      req.TransactionID,
		"PartyA":             m.HeadOffice,
		"IdentifierType":     2,
		"Remarks":            "Checking Transaction Status",
		"QueueTimeOutURL":    m.CallbackURL,
		"ResultURL":          m.CallbackURL,
		"Occassion":          "",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling payload %s", err.Error())
		return nil, err
	}
	url := fmt.Sprintf("%smpesa/transactionstatus/v1/query", m.MpesaURL)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request %s", err.Error())
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+m.AccessToken)
	request.Header.Set(content, contentTypeJSON)

	client := &http.Client{}
	res, err := client.Do(request)
	if err != nil {
		log.Printf("Error making request %s", err.Error())
		return nil, err
	}
	defer res.Body.Close()

	var result dtos.MpesaTransactionStatusRequest
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding response %s", err.Error())
		return nil, err
	}
	log.Printf("MPESA Transaction Status Response: %+v", result)
	return &result, nil
}

func MpesaCallbackHandler(w http.ResponseWriter, r *http.Request) {
	var callback dtos.MpesaResultResponse
	//log recieved callback
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to decode MPESA callback: %v", err)
		return
	}
	log.Printf("Received MPESA callback: %s", string(bodyBytes))
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
		log.Printf("Failed to decode MPESA callback: %v", err)
		return
	}

	trxID := callback.Result.TransactionID
	if trxID == "" {
		log.Printf("Missing TransactionID in callback")
		return
	}

	// Save as "done" in Redis
	finalObj := map[string]interface{}{
		"status": "done",
		"data":   callback,
	}
	utils.SetCache("mpesa_status:"+trxID, finalObj, 2*time.Minute)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Transaction status callback received"}`))
}
