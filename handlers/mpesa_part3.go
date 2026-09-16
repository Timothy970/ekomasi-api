package handlers

import (
	"bytes"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler for the MPesa callback
func HandleMpesaBalance(c *gin.Context) {
	log.Printf("MPESA CALLBACK FROM IP: %s", c.Request.RemoteAddr)

	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	client, err := NewMpesaClient()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create mpesa client")
		return
	}
	result, err := client.FetchPayBillBalance()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{

				Module:      "Payments",
				Description: "Failed to fetch MPESA PayBill balance",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("Failed to fetch MPESA PayBill balance %s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	resultJSON, _ := json.Marshal(result)
	log.Printf("callback body:::::%v", string(resultJSON))
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "MPESA PayBill balance fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   result,
		Message:   "PayBill balance fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

func HandleMpesaBalanceCallback(c *gin.Context) {
	log.Printf("MPESA BALANCE CALLBACK FROM IP: %s", c.Request.RemoteAddr)
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "failed to read body")
		return
	}
	log.Printf("balance callback body:::::%v", string(bodyBytes))

	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
}

type MpesaMoneyReturnRequest struct {
	OrderID     string `json:"order_id" validate:"required"`
	PhoneNumber string `json:"phone_number" validate:"required"`
}

func HandleMpesaReturnCallback(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "failed to read body")
		return
	}
	var callback dtos.MpesaTransactionRufundResponse
	if err := json.Unmarshal(bodyBytes, &callback); err != nil {
		c.String(http.StatusBadRequest, "failed to parse callback")
		return
	}
	phoneNumber := ""
	for _, param := range callback.Result.ResultParameters.ResultParameter {
		if param.Key == "ReceiverPartyPublicName" {
			phoneNumber = param.Value.(string)
			break
		}
	}
	if callback.Result.ResultCode == 0 {
		transaction := dtos.TransactionsList{
			OrderID:              nil,
			TransactionReference: "REFUND-" + "OriginalTransactionID-" + callback.Result.OriginatorConversationID + "-" + "ConversationID-" + callback.Result.ConversationID,
			MpesaReference:       &callback.Result.TransactionID,
			PhoneNumber:          &phoneNumber,
			Amount:               0,
			Status:               "SUCCESS",
			PaymentMethod:        "MPESA",
		}
		log.Printf("Refund with transaction details %v successful", transaction)
		err := models.InsertTransaction(models.DB, &transaction)
		if err != nil {
			log.Printf("Failed to insert transaction log for MPESA money return: %v", err)
		}
		//INSERT INTO TRASACTION LOG WITH STATUS REFUNDED
	}
	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
}

func HandleMpesaMoneyReturn(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	//check if user is admin
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", "payments.refund")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[MpesaMoneyReturnRequest](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Accounts") {
		return
	}
	order, err := models.GetOrderByID(models.DB, req.OrderID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
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
		c.String(http.StatusInternalServerError, "failed to create mpesa client")
		return
	}

	result, err := client.HandleMoneyReturn(order.TotalAmount, req.PhoneNumber)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to handle MPESA money return",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("Failed to handle MPESA money return %s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	if result.ResultCode != "0" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "MPESA money return failed",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("MPESA money return failed: %s", result.ResultDesc),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	//mark the order as refunded and restock items
	err = models.HandleMpesaMoneyReturnRefunds(models.DB, *order)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to process refunds for MPESA money return",
				Code:        http.StatusInternalServerError,
			},
			Message:   fmt.Sprintf("Failed to process refunds for MPESA money return %s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "MPESA money return handled successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "MPESA money return handled successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}

func (m *MpesaClient) HandleMoneyReturn(amount float64, phoneNumber string) (MpesaMoneyReturnResponse, error) {
	payload := map[string]any{
		"OriginatorConversationID": randString(24),
		"InitiatorName":            m.InitiatorName,
		"SecurityCredential":       m.SecurityCredential,
		"CommandID":                "BusinessPayment",
		"Amount":                   amount,
		"PartyA":                   m.ShortCode,
		"PartyB":                   phoneNumber,
		"Remarks":                  "Payment Return for order to phone number " + phoneNumber,
		"QueueTimeOutURL":          m.ReturnURL,
		"ResultURL":                m.ReturnURL,
		"Occasion":                 "",
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

	var result map[string]any
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
