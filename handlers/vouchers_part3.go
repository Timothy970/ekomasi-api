package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func GetUserVoucherHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: voucherNoUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	voucherID := c.Param("voucher_id")

	voucher, err := models.GetUserVoucherByID(models.DB, voucherID, authuser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher with ID " + voucherID + " for user with ID " + fmt.Sprint(authuser.ID),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: voucherWithID + voucherID + " fetched successfully for user with ID " + fmt.Sprint(authuser.ID),
			Code:        http.StatusOK,
		},
		Payload:   voucher,
		Message:   "Voucher fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListUserVoucherHandler retrieves all vouchers owned by the authenticated user.
// Returns paginated list of available gift cards and promotional codes for the user.
// Essential for user wallet/profile view of their available discounts.
//
// @Summary      List user Vouchers
// @Description  Retrieve a paginated list of all vouchers owned by the authenticated user
// @Tags         Users
// @Produce      json
// @Param        page  query     int                      false  "Page number (default: 1)"
// @Param        size  query     int                      false  "Page size (default: 10)"
// @Success      200   {object}  map[string]any   "User vouchers list"
// @Failure      401   {object}  dtos.ErrorResponse       "User authentication required"
// @Security     BearerAuth
// @Router       /api/user/vouchers [get]
func ListUserVoucherHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: voucherNoUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	page, limit := parsePagination(c.Query("page"), c.Query("size"))
	voucher, pagination, err := models.GetUserVouchers(models.DB, authuser.ID, page, limit)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers for user with ID " + fmt.Sprint(authuser.ID),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Vouchers fetched successfully for user with ID " + fmt.Sprint(authuser.ID),
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"vouchers": voucher, "pagination": pagination},
		Message:   "Vouchers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// BuyVoucherHandler allows a user to purchase a new voucher.
// Users can buy vouchers for themselves or as gifts for others (sent via email).
// Triggers payment processing (e.g., M-Pesa) before voucher activation.
//
// @Summary      Purchase a voucher
// @Description  Process a request to buy a new voucher, initiating payment
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        voucher_purchase  body      dtos.BuyVoucherData    true  "Voucher purchase details"
// @Success      201               {object}  map[string]any "Purchase initiated, payment pending"
// @Failure      400               {object}  dtos.ErrorResponse     "Invalid purchase request"
// @Failure      401               {object}  dtos.ErrorResponse     "User authentication required"
// @Security     BearerAuth
// @Router       /api/user/vouchers/buy [post]
func BuyVoucherHandler(c *gin.Context) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(c.Request)

	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: voucherNoUserFound,
				Code:        http.StatusUnauthorized,
			},
			Message:   voucherNoUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.BuyVoucherData](c, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Vouchers") {
		return
	}
	// create voucher data
	var voucher dtos.Voucher
	voucher.Amount = req.Amount
	voucher.DesignID = &req.DesignID
	active := "scheduled"
	voucher.Status = &active

	deliveryTime := models.StringToTime(req.DeliveryTime)
	//check delivery time is in the past (but allow today's date)
	today := time.Now().Truncate(24 * time.Hour)
	if deliveryTime.Before(today) {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Delivery time cannot be in the past",
				Code:        http.StatusBadRequest,
			},
			Message:   "Delivery time cannot be in the past",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	expiryEnv := os.Getenv("VOUCHER_EXPIRY_DATE")
	if expiryEnv == "" {
		// Add 90 days from the delivery date
		expiryDate := deliveryTime.AddDate(0, 0, 90)
		voucher.ExpiryDate = expiryDate.Format("2006-01-02 15:04:05")
	} else {
		// Use expiry from environment variable
		expiryTime := models.StringToTime(expiryEnv)
		voucher.ExpiryDate = expiryTime.Format("2006-01-02 15:04:05")
	}

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

	voucherID, err := models.AddNewVoucher(tx, voucher, authuser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	//insert into voucher purchases
	err = models.InsertIntoVoucherPurchases(tx, *req, authuser.ID, voucherID)
	if err != nil {
		// Purchase recording failed
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to record voucher purchase",
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
	voucherOrderID, err := models.CreateVoucherOrder(tx, req.Amount, voucherID, req.PaymentMethod)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher order",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
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
			Description: "Voucher created and added successfully",
			Code:        http.StatusCreated,
		},
		Payload: map[string]any{
			"voucher_order_id": voucherOrderID,
		},
		Message:   "Voucher created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
