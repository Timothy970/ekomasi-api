package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func BuyVoucherUpdateHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)

	req, ok := DecodeRequestBody[dtos.BuyVoucherData](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Vouchers") {
		return
	}
	voucherID := c.Param("voucher_id")
	// create voucher data
	amount := req.Amount
	status := "scheduled"
	// Start transaction
	tx, err := models.DB.Begin()
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to start transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	defer tx.Rollback()

	err = models.ValidateDesignID(tx, req.DesignID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Invalid Design ID",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if req.Amount > 0 {
		err := models.UpdateVoucher(tx, voucherID, amount, status, req.DesignID)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: "Failed to update voucher",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}
	//insert into voucher purchases
	err = models.UpdateVoucherPurchases(tx, *req, voucherID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher purchase",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//create voucher order
	voucherOrderID, err := models.UpdateVoucherPurchaseAmount(tx, req.Amount, voucherID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher purchase amount",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	if req.Amount > 0 || req.PaymentMethod != "none" {
		err = voucherPaymentProcessor(tx, req.PaymentMethod, voucherOrderID, req.PhoneNumber, req.Amount, voucherID)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: "Failed to process voucher payment",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to commit transaction",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher updated and added successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"voucher_order_id": voucherOrderID,
		},
		Message:   "Voucher updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
func voucherPaymentProcessor(db models.DBExecutor, paymentMethod string, voucherOrderID, phoneNumber string, amount float64, voucherID string) error {
	switch paymentMethod {
	//where method is mpesa or empty use mpesa
	case "mpesa", "":
		// Initiate Mpesa payment
		err := HandleMpesaVoucherPayment(db, voucherOrderID, phoneNumber, amount)
		if err != nil {
			return err
		}
	case "cash":
		// For cash payments, we can directly mark the order as paid and activate the voucher without external processing to active
		err := models.MarkVoucherAsPaid(db, voucherID)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported payment method: %s", paymentMethod)
	}
	return nil
}

// RedeemVoucherHandler allows a user to redeem a voucher to their account.
// User must be authenticated and provide a valid voucher code.
// Transfers voucher ownership to the user for future use on purchases.
//
// @Summary      Redeem voucher
// @Description  Redeem a voucher code to user's account
// @Tags         Vouchers
// @Accept       json
// @Produce      json
// @Param        voucher  body      dtos.RedeemVoucherRequest  true  "Voucher redemption details"
// @Success      200      {object}  map[string]any       "Voucher redeemed successfully"
// @Failure      400      {object}  dtos.ErrorResponse         "Invalid code or redemption failed"
// @Failure      401      {object}  dtos.ErrorResponse         "User authentication required"
// @Security     BearerAuth
// @Router       /api/vouchers/redeem [post]
func RedeemVoucherHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Get authenticated user from context
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		// User not authenticated
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "User not validated or unauthorized",
				Code:        http.StatusInternalServerError,
			},
			Message:   voucherNoUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Decode and parse JSON request body with voucher code
	req, ok := DecodeRequestBody[dtos.RedeemVoucherRequest](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate voucher code format
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Vouchers") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Redeem voucher code to user's account
	_, err := models.RedeemVoucher(models.DB, req.Code, authuser.ID)
	if err != nil {
		// Redemption failed (invalid code, already redeemed, expired, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to redeem voucher",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate voucher caches to reflect redemption status
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher redeemed successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher redeemed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// StartVoucherEmailScheduler initializes a background scheduler for sending voucher emails.
// Runs at specified intervals to process scheduled voucher deliveries.
// The scheduler can run indefinitely or for a specified number of iterations.
func StartVoucherEmailScheduler(interval time.Duration, repeat int) {
	// Create ticker for periodic execution
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		// Execute on each tick
		for range ticker.C {
			// Process and send scheduled voucher emails
			SendBoughtForVoucherEmails()

			// Decrement repeat counter if limited execution
			if repeat > 0 {
				repeat--
				if repeat == 0 {
					// Stop scheduler after specified iterations
					return
				}
			}
		}
	}()
}

// SendBoughtForVoucherEmails processes and sends scheduled voucher delivery emails.
// Fetches vouchers with unsent emails and sends them to recipients.
// Updates email sent status after successful delivery.
func SendBoughtForVoucherEmails() {
	// Fetch all users with vouchers pending email delivery
	users, err := models.GetUsersWithUnsentVoucherEmails(models.DB)
	if err != nil {
		fmt.Printf("error fetching users with unsent voucher emails: %v", err)
		return
	}
	// Process each voucher email
	for _, u := range users {
		// Send email if recipient email is provided
		if u.ToEmail != "" {
			// Generate email subject and HTML body
			subject, htmlBody := utils.SendVoucherEmail(u)
			// Send email notification
			notification.SendEmail(u.ToEmail, subject, htmlBody)

			// Mark voucher email as sent in database
			err := models.MarkVoucherEmailAsSent(models.DB, u.VoucherID)
			if err != nil {
				log.Printf("error marking voucher email as sent for user: %v", err)
			}
		}
	}
}
