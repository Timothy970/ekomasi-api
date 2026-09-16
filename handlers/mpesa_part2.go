package handlers

import (
	"bytes"
	"crypto/rand"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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
func HandleMpesaCallback(c *gin.Context) {
	//log the IP address of the caller
	log.Printf("MPESA CALLBACK FROM IP: %s", c.Request.RemoteAddr)
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "failed to read body")
		return
	}
	log.Printf("callback body:::::%v", string(bodyBytes))

	var callback dtos.STKCallbackRequest
	if err := json.Unmarshal(bodyBytes, &callback); err != nil {
		c.String(http.StatusBadRequest, "failed to parse callback")
		return
	}

	stk := callback.Body.StkCallback
	logCallbackInfo(stk.CheckoutRequestID, stk.ResultDesc)

	if stk.ResultCode == 0 {
		handleSuccessfulPayment(c, callback)
		return
	}

	handleFailedPayment(c, callback)

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
	return models.UpdateMpesaReceiptNumber(models.DB, mpesaCode, orderID)
}

// logCallbackInfo logs callback metadata
func logCallbackInfo(checkoutID, resultDesc string) {
	log.Printf("MPESA CALLBACK RECEIVED:\n- CheckoutRequestID: %s\n- Result: %s\n- Time: %s\n",
		checkoutID, resultDesc, time.Now().Format(time.RFC3339))
}

// handleSuccessfulPayment processes successful Mpesa payments
func handleSuccessfulPayment(c *gin.Context, callback dtos.STKCallbackRequest) {
	stk := callback.Body.StkCallback
	amount, mpesaCode, phone := extractMetadata(stk.CallbackMetadata.Item)

	log.Printf("SUCCESSFUL PAYMENT:\n- Phone: %s\n- Amount: %.2f\n- Code: %s\n", phone, amount, mpesaCode)

	deliveryID, orderID, orderType, err := models.UpdateStkResponse(models.DB, callback.Body.StkCallback.CheckoutRequestID, "SUCCESS")
	if err != nil {
		log.Printf("error updating STK response: %v", err)
	}
	if orderID == "" && deliveryID == "" {
		log.Printf("orderID and deliveryID are both empty, trying sending the callback to development environment")
	}
	processOrderUpdate(orderType, deliveryID, orderID, "COMPLETED")
	//update transaction log
	err = models.UpdateTransactionStatus(models.DB, orderID, "COMPLETED")
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
	if c != nil {
		c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
	}
}

// handleFailedPayment logs failed payments and responds OK
func handleFailedPayment(c *gin.Context, callback dtos.STKCallbackRequest) {
	resultCode := callback.Body.StkCallback.ResultCode
	resultDesc := callback.Body.StkCallback.ResultDesc
	checkoutRequestID := callback.Body.StkCallback.CheckoutRequestID
	deliveryID, orderID, orderType, err := models.UpdateStkResponse(models.DB, checkoutRequestID, "FAILED")
	if err != nil {
		log.Printf("error updating STK response: %v", err)
	}
	if orderID == "" && deliveryID == "" {
		log.Printf("orderID and deliveryID are both empty, trying sending the callback to development environment")
	}
	processOrderUpdate(orderType, deliveryID, orderID, "FAILED")
	//update transaction log
	err = models.UpdateTransactionStatus(models.DB, orderID, "FAILED")
	if err != nil {
		log.Printf("error updating transaction status: %v", err)
	}
	log.Printf("FAILED PAYMENT:\n- Code: %d\n- Desc: %s\n", resultCode, resultDesc)
	if c != nil {
		c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Callback received"})
	}
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
		err = models.UpdateVoucherOrderTables(models.DB, orderID, status)
		if err != nil {
			log.Printf("error updating voucher order: %v", err)
		}
		utils.SendToUser(fmt.Sprintf("%s:voucher_order", orderID), buildPaymentSuccessPayload(orderID, nil, status))
	default:
		err = models.UpdateDeliveryOrderTables(models.DB, deliveryID, orderID, status)
		if err != nil {
			log.Printf("error updating delivery order: %v", err)
		}
		utils.SendToUser(fmt.Sprintf("%s:%s", orderID, deliveryID), buildPaymentSuccessPayload(orderID, deliveryID, status))
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

func HandleMpesaVoucherPayment(db models.DBExecutor, orderID, phoneNumber string, amount float64) error {

	req := &dtos.MpesaRequest{
		OrderID:     orderID,
		Phone:       phoneNumber,
		Amount:      int(amount),
		DeliveryID:  "",
		Reference:   "EKOMASI VOUCHER -" + orderID,
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
	err = models.StoreStkResponse(db, response, *req)
	if err != nil {
		return err
	}
	return nil
}

func (m *MpesaClient) FetchPayBillBalance() (map[string]any, error) {

	payload := map[string]interface{}{
		"Initiator":          m.InitiatorName,
		"SecurityCredential": m.SecurityCredential,
		"CommandID":          "AccountBalance",
		"PartyA":             m.ShortCode,
		"IdentifierType":     "4",
		"Remarks":            "balance",
		"QueueTimeOutURL":    m.BalanceURL,
		"ResultURL":          m.BalanceURL,
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
