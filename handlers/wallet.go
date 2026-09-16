package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type WalletTopupRequest struct {
	PhoneNumber string  `json:"phone_number" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
}

type WalletCheckoutPaymentRequest struct {
	OrderID     string  `json:"order_id" validate:"required"`
	PhoneNumber string  `json:"phone_number" validate:"required"`
	TotalAmount float64 `json:"total_amount" validate:"required,gt=0"`
}

// GetWalletBalanceHandler returns user's current wallet balance and history
func GetWalletBalanceHandler(c *gin.Context) {
	start := time.Now()
	userPhone := c.Query("phone")
	if userPhone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phone parameter is required"})
		return
	}

	var balance float64
	var walletID int
	err := models.DB.QueryRow(`SELECT id, balance FROM user_wallets WHERE user_phone = ?`, userPhone).Scan(&walletID, &balance)
	if err != nil {
		balance = 0.0
	}

	rows, _ := models.DB.Query(`SELECT id, type, amount, reference, description, created_at FROM wallet_transactions WHERE user_phone = ? ORDER BY id DESC LIMIT 20`, userPhone)
	defer rows.Close()

	var transactions []map[string]any
	for rows.Next() {
		var id int
		var txType, ref, desc, createdAt string
		var amount float64
		rows.Scan(&id, &txType, &amount, &ref, &desc, &createdAt)
		transactions = append(transactions, map[string]any{
			"id":          id,
			"type":        txType,
			"amount":      amount,
			"reference":   ref,
			"description": desc,
			"created_at":  createdAt,
		})
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Wallet",
			Description: "Wallet balance fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"user_phone":   userPhone,
			"balance":      balance,
			"transactions": transactions,
		},
		Message:   "Wallet details retrieved",
		TimeTaken: time.Since(start),
		Function:  "GetWalletBalanceHandler",
		Request:   c.Request,
	})
}

// TopupWalletMpesaHandler triggers an M-Pesa STK Push to directly top up the user's wallet balance
func TopupWalletMpesaHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	req, ok := DecodeRequestBody[WalletTopupRequest](c, requestSummary, start)
	if !ok {
		return
	}

	reference := fmt.Sprintf("TOPUP-%d", time.Now().Unix())
	mpesaReq := dtos.MpesaRequest{
		Phone:       req.PhoneNumber,
		Amount:      int(req.Amount),
		Reference:   reference,
		Description: "Wallet Top-up",
		OrderID:     reference,
	}

	client, err := NewMpesaClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize M-Pesa client"})
		return
	}

	response, err := client.LipaNaMpesaOnline(mpesaReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "M-Pesa STK Push failed: " + err.Error()})
		return
	}

	checkoutID := fmt.Sprintf("%v", response["CheckoutRequestID"])
	log.Printf("[Wallet Topup] STK Push initiated for %s: CheckoutRequestID=%s", req.PhoneNumber, checkoutID)

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Wallet",
			Description: "Wallet top-up STK push initiated",
			Code:        http.StatusOK,
		},
		Payload: gin.H{
			"checkout_request_id": checkoutID,
			"reference":           reference,
		},
		Message:   "STK push prompt sent to your phone for wallet topup",
		TimeTaken: time.Since(start),
		Function:  "TopupWalletMpesaHandler",
		Request:   c.Request,
	})
}

// PayWithWalletOrSplitHandler uses wallet balance first. If balance is less than TotalAmount,
// it deducts full wallet balance and initiates M-Pesa STK push for the remaining deficit amount.
func PayWithWalletOrSplitHandler(c *gin.Context) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(c.Request)
	req, ok := DecodeRequestBody[WalletCheckoutPaymentRequest](c, requestSummary, start)
	if !ok {
		return
	}

	var currentBalance float64
	var walletID int
	err := models.DB.QueryRow(`SELECT id, balance FROM user_wallets WHERE user_phone = ?`, req.PhoneNumber).Scan(&walletID, &currentBalance)

	if err != nil || currentBalance <= 0 {
		// Wallet has no balance -> Full payment via M-Pesa STK Push
		mpesaReq := dtos.MpesaRequest{
			Phone:       req.PhoneNumber,
			Amount:      int(req.TotalAmount),
			Reference:   "EKOMASI-" + req.OrderID,
			Description: "Full Payment for Order " + req.OrderID,
			OrderID:     req.OrderID,
		}
		client, _ := NewMpesaClient()
		stkResp, _ := client.LipaNaMpesaOnline(mpesaReq)
		checkoutID := fmt.Sprintf("%v", stkResp["CheckoutRequestID"])

		c.JSON(http.StatusOK, gin.H{
			"payment_type":        "FULL_MPESA",
			"wallet_used":         0.0,
			"mpesa_amount":        req.TotalAmount,
			"checkout_request_id": checkoutID,
			"message":             "Wallet balance insufficient. Full payment STK Push sent to phone.",
		})
		return
	}

	if currentBalance >= req.TotalAmount {
		// Wallet covers full amount -> Deduct full amount from wallet
		models.DB.Exec(`UPDATE user_wallets SET balance = balance - ? WHERE id = ?`, req.TotalAmount, walletID)
		models.DB.Exec(`INSERT INTO wallet_transactions (wallet_id, user_phone, type, amount, reference, description) VALUES (?, ?, 'DEBIT', ?, ?, ?)`,
			walletID, req.PhoneNumber, req.TotalAmount, req.OrderID, "Full Order Payment from Wallet")

		c.JSON(http.StatusOK, gin.H{
			"payment_type": "FULL_WALLET",
			"wallet_used":  req.TotalAmount,
			"mpesa_amount": 0.0,
			"message":      "Order fully paid using in-app wallet balance!",
		})
		return
	}

	// Split Payment Case: Deduct full current balance, prompt M-Pesa STK push for remaining deficit
	walletDeduction := currentBalance
	mpesaDeficit := req.TotalAmount - currentBalance

	// 1. Deduct wallet balance to 0
	models.DB.Exec(`UPDATE user_wallets SET balance = 0 WHERE id = ?`, walletID)
	models.DB.Exec(`INSERT INTO wallet_transactions (wallet_id, user_phone, type, amount, reference, description) VALUES (?, ?, 'DEBIT', ?, ?, ?)`,
		walletID, req.PhoneNumber, walletDeduction, req.OrderID, fmt.Sprintf("Split Payment: Wallet portion for Order %s", req.OrderID))

	// 2. Trigger M-Pesa STK push for the remaining deficit
	mpesaReq := dtos.MpesaRequest{
		Phone:       req.PhoneNumber,
		Amount:      int(mpesaDeficit),
		Reference:   "EKOMASI-" + req.OrderID,
		Description: fmt.Sprintf("Split Payment Deficit for Order %s", req.OrderID),
		OrderID:     req.OrderID,
	}
	client, _ := NewMpesaClient()
	stkResp, _ := client.LipaNaMpesaOnline(mpesaReq)
	checkoutID := fmt.Sprintf("%v", stkResp["CheckoutRequestID"])

	c.JSON(http.StatusOK, gin.H{
		"payment_type":        "SPLIT_WALLET_MPESA",
		"wallet_used":         walletDeduction,
		"mpesa_amount":        mpesaDeficit,
		"checkout_request_id": checkoutID,
		"message":             fmt.Sprintf("KES %.2f paid from wallet. M-Pesa STK Push sent for remaining KES %.2f deficit.", walletDeduction, mpesaDeficit),
	})
}
