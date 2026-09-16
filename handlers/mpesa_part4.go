package handlers

import (
	"bytes"
	"ekomasi_backend/dtos"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func HandleMpesaTransactionStatus(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	req, ok := DecodeRequestBody[dtos.MpesaTransactionStatus](c, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		return
	}
	client, err := NewMpesaClient()
	if err != nil {
		utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to initialize MPESA client",
				Code:        http.StatusOK,
			},
			Payload:   nil,
			Message:   fmt.Sprintf("Failed to initialize MPESA client %s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	response, err := client.CheckMpesaTransactionStatus(*req)
	if err != nil {
		utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to initiate MPESA payment",
				Code:        http.StatusOK,
			},
			Payload:   nil,
			Message:   "Failed to get transaction status!",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	if response.ResponseCode != "0" {
		log.Printf("MPESA transaction status fetch failed: %s", response.ResponseDescription)
		utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "MPESA transaction status fetch failed",
				Code:        http.StatusOK,
			},
			Payload:   nil,
			Message:   "Failed to get transaction status!",
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
			Description: "MPESA transaction status waiting for callback",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Transaction status waiting for callback!",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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

	payload := map[string]any{
		"Initiator":          m.InitiatorName,
		"SecurityCredential": m.SecurityCredential,
		"CommandID":          "TransactionStatusQuery",
		"TransactionID":      req.TransactionID,
		"PartyA":             m.ShortCode,
		"IdentifierType":     4,
		"Remarks":            "Checking Transaction Status",
		"QueueTimeOutURL":    m.StatusURL,
		"ResultURL":          m.StatusURL,
		"Occassion":          "Payment Status",
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
	log.Printf("response code: %s", result.ResponseCode)
	log.Printf("response code type: %T", result.ResponseCode)
	return &result, nil
}

func MpesaCallbackHandler(c *gin.Context) {
	var callback dtos.MpesaResultResponse
	//log recieved callback
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to decode MPESA callback: %v", err)
		return
	}
	log.Printf("Received MPESA callback: %s", string(bodyBytes))
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	if err := c.ShouldBindJSON(&callback); err != nil {
		log.Printf("Failed to decode MPESA callback: %v", err)
		return
	}

	trxID := callback.Result.TransactionID
	if trxID == "" {
		log.Printf("Missing TransactionID in callback")
		return
	}

	// Save as "done" in Redis
	finalObj := map[string]any{
		"status": "done",
		"data":   callback,
	}
	utils.SetCache("mpesa_status:"+trxID, finalObj, 2*time.Minute)
	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Transaction status callback received"})
}

// handler for mpesa transaction status callback// Handler for the MPesa callback
func HandleTransactionStatusCallback(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "failed to read body")
		return
	}

	var callback dtos.TransactionStatusSafaricomResponse
	if err := json.Unmarshal(bodyBytes, &callback); err != nil {
		c.String(http.StatusBadRequest, "failed to parse callback")
		return
	}
	message := map[string]any{}
	if callback.Result.ResultCode == 0 {
		message["status"] = "success"
		message["message"] = "Transaction status fetched successfully"
		for _, param := range callback.Result.ResultParameters.ResultParameter {

			switch param.Key {

			case "ReceiptNo":
				if receipt, ok := param.Value.(string); ok {
					message["transaction_id"] = receipt
				}

			case "Amount":
				if amountStr, ok := param.Value.(string); ok {
					message["amount"] = amountStr
				}

			case "TransactionStatus":
				if status, ok := param.Value.(string); ok {
					message["transaction_status"] = status
				}
			}
		}
		log.Printf("message to deliver %v", message)
		key := message["transaction_id"].(string)
		utils.SendToUser(key, message)
	}
}
