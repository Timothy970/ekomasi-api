package handlers

import (
	"bytes"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	contentTypeJSON = "application/json"
	content         = "Content-Type"
)

func HandleMpesaPayment(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	req, ok := DecodeRequestBody[dtos.MpesaRequest](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}

	order, err := models.GetOrderByID(models.DB, req.OrderID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get order for MPESA payment",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// computedAmount := int(order.TotalAmount) - int(order.TotalDiscount)
	//for testing we can set the computed amount to 1 to avoid issues with zero amount payments in M-Pesa sandbox
	computedAmount := 1
	if computedAmount < 0 {
		log.Printf("negative MPESA payment amount computed for order %s: total=%v, discount=%v, computedAmount=%d; clamping to 0",
			order.OrderID, order.TotalAmount, order.TotalDiscount, computedAmount)
		req.Amount = 0
	} else {
		req.Amount = computedAmount
	}
	req.DeliveryID = order.DeliveryID
	req.Reference = "EKOMASI - " + order.OrderID
	req.Description = fmt.Sprintf("Payment for order %s", order.OrderID)
	if req.Amount > 0 {
		client, err := NewMpesaClient()
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: "Failed to initialize MPESA client",
					Code:        http.StatusInternalServerError,
				},
				Message:   fmt.Sprintf("Failed to initialize MPESA client %s", err),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return
		}

		response, err := client.LipaNaMpesaOnline(*req)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: "Failed to initiate MPESA payment",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return
		}
		err = models.StoreStkResponse(models.DB, response, *req)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: "Failed to store MPESA payment request",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return
		}
	}
	err = storeTransactionLog(models.DB, *req)
	if err != nil {
		log.Printf("Failed to store transaction log: %v", err)
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "MPESA payment request initiated successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Payment request initiated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
	if req.Amount == 0 {
		//if the amount is zero, we can skip calling M-Pesa and directly mark the order as paid and send the callback
		err := models.MarkOrderAsPaid(models.DB, req.OrderID)
		if err != nil {
			log.Printf("Failed to mark order as paid: %v", err)
		}
	}
}

func storeTransactionLog(db models.DBExecutor, req dtos.MpesaRequest) error {
	logEntry := &dtos.TransactionsList{
		OrderID:              &req.OrderID,
		TransactionReference: req.Reference,
		PhoneNumber:          &req.Phone,
		Amount:               float64(req.Amount),
		Status:               "PENDING",
		PaymentMethod:        "MPESA",
	}
	return models.InsertTransaction(db, logEntry)
}

func RegisterMpesaRoutesHandler(c *gin.Context) {
	client, err := NewMpesaClient()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if err := client.RegisterURLs(); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusOK, "M-Pesa URLs registered successfully")
}

type MpesaClient struct {
	ConsumerKey        string
	ConsumerSecret     string
	ShortCode          string
	Passkey            string
	CallbackURL        string
	BalanceURL         string
	AccessToken        string
	MpesaURL           string
	InitiatorName      string
	InitiatorPassword  string
	SecurityCredential string
	ReturnURL          string
	StatusURL          string
}

func NewMpesaClient() (*MpesaClient, error) {
	InitiatorPassword := os.Getenv("MPESA_INITIATOR_PASSWORD")
	client := &MpesaClient{
		ConsumerKey:        os.Getenv("MPESA_CONSUMER_KEY"),
		ConsumerSecret:     os.Getenv("MPESA_CONSUMER_SECRET"),
		ShortCode:          os.Getenv("MPESA_SHORTCODE"),
		Passkey:            os.Getenv("MPESA_PASSKEY"),
		CallbackURL:        os.Getenv("MPESA_CALLBACK_URL"),
		BalanceURL:         os.Getenv("MPESA_BALANCE_URL"),
		MpesaURL:           os.Getenv("MPESA_SEND_URL"),
		InitiatorName:      os.Getenv("MPESA_INITIATOR_NAME"),
		InitiatorPassword:  InitiatorPassword,
		SecurityCredential: os.Getenv("MPESA_SECURITY_CREDENTIALS"),
		ReturnURL:          os.Getenv("MPESA_RETURN_URL"),
		StatusURL:          os.Getenv("MPESA_STATUS_URL"),
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

	var result map[string]any
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

func (m *MpesaClient) LipaNaMpesaOnline(paymentRequest dtos.MpesaRequest) (map[string]any, error) {
	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString([]byte(m.ShortCode + m.Passkey + timestamp))
	log.Printf("Payment request::::%v", paymentRequest)
	payload := map[string]any{
		"BusinessShortCode": m.ShortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerPayBillOnline",
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
	log.Printf("Payment url to use:::%s", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.AccessToken)
	req.Header.Set(content, contentTypeJSON)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request::: %s", err)
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding response::: %s", err)
		return nil, err
	}

	return result, nil
}
