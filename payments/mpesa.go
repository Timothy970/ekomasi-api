package payments

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"bytes"
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

func HandleMpesaPayment(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	var req dtos.MpesaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "Invalid request",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	client, err := NewMpesaClient()
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
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
			Code:      http.StatusBadRequest,
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
			Code:      http.StatusBadRequest,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   response,
		Message:   "Payment request initiated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
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
	req.Header.Set("Content-Type", "application/json")

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
	req.Header.Set("Content-Type", "application/json")

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

// Handler for the MPesa callback
func HandleMpesaCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("************************callback hit******************")
	// start := time.Now()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	log.Printf("callback body:::::%v", string(bodyBytes))
	var callback dtos.STKCallbackRequest
	log.Printf("************************callback marshalling******************")

	if err := json.Unmarshal(bodyBytes, &callback); err != nil {
		http.Error(w, "failed to parse callback", http.StatusBadRequest)
		return
	}
	log.Printf("************************callback finished******************")

	// Pretty-print the struct if needed
	callbackJSON, _ := json.MarshalIndent(callback, "", "  ")
	log.Printf("********** Parsed Callback Struct:\n%s", string(callbackJSON))

	stk := callback.Body.StkCallback

	log.Printf("MPESA CALLBACK RECEIVED:\n- CheckoutRequestID: %s\n- Result: %s\n- Time: %s\n",
		stk.CheckoutRequestID, stk.ResultDesc, time.Now().Format(time.RFC3339))

	if stk.ResultCode == 0 {
		// Success - extract info from metadata
		var amount float64
		var mpesaCode, phone string

		for _, item := range stk.CallbackMetadata.Item {
			switch item.Name {
			case "Amount":
				amount = item.Value.(float64)
			case "MpesaReceiptNumber":
				mpesaCode = item.Value.(string)
			case "PhoneNumber":
				phone = fmt.Sprintf("%.0f", item.Value.(float64)) // from float to string
			}
		}

		// Log or store success transaction
		log.Printf("SUCCESSFUL PAYMENT:\n- Phone: %s\n- Amount: %.2f\n- Code: %s\n", phone, amount, mpesaCode)
		deliveryID, orderID, orderType, err := models.UpdateStkResponse(callback, "SUCCESS")
		log.Printf("orderType: %s, deliveryID: %s, orderID: %s, err: %v", orderType, deliveryID, orderID, err)
		if err != nil {
			log.Printf("%v", err)
		}
		switch strings.ToLower(orderType) {
		case "voucher":
			//update delivery and order tables
			err = models.UpdateVoucherOrderTables(orderID)
			if err != nil {
				log.Printf("%v", err)
			}
			utils.SendToUser(
				"",
				orderID,
				"voucher_order",
				map[string]interface{}{
					"event":       "payment_success",
					"message":     "Your payment was successful!",
					"order_id":    orderID,
					"delivery_id": nil,
				},
			)
		default:
			//update delivery and order tables
			err = models.UpdateDeliveryOrderTables(deliveryID, orderID)
			if err != nil {
				log.Printf("%v", err)
			}
			utils.SendToUser(
				"",
				orderID,
				deliveryID,
				map[string]interface{}{
					"event":       "payment_success",
					"message":     "Your payment was successful!",
					"order_id":    orderID,
					"delivery_id": deliveryID,
				},
			)
		}
		// Respond OK
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
		return
	}

	// Log failed payment
	log.Printf("❌ FAILED PAYMENT:\n- Code: %d\n- Desc: %s\n", stk.ResultCode, stk.ResultDesc)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Callback received"}`))
}
