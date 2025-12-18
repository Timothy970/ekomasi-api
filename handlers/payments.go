package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

var paymentWithID = "Payment with ID "

// @Summary  Create Payment
// @Description Create Payment
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/payments [POST]
func CreatePaymentHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Payments")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.Payment](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}
	if err := models.CreatePayment(*req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to create payment",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("payments_")
	utils.DeleteCacheByPrefix("payments_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Payment created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get payment by ID
// @Summary List Payment
// @Description List Payment
// @Tags Payments
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/pyaments/{payment_id} [GET]
func GetPaymentByIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	id := mux.Vars(r)["payment_id"]
	payment, err := models.GetPaymentByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get payment by ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentWithID + id + " has been fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   payment,
		Message:   "Payment fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// List all payments
// @Summary List payments
// @Description List payments
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/payments [GET]
func ListPaymentsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyPayments := fmt.Sprintf("payments_%d_size_%d", page, limit)
	cacheKeyPagination := fmt.Sprintf("payments_pagination_%d_size_%d", page, limit)
	var payments []dtos.Payment
	var cachedPayemnts []dtos.Payment
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyPayments, &cachedPayemnts)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if len(cachedPayemnts) == 0 {
		var err error
		payments, pagination, err = models.ListPayments(page, limit)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: "Failed to list payments",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKeyPayments, payments)
		_ = utils.SetCache(cacheKeyPagination, pagination)
	} else {
		payments = cachedPayemnts
		pagination = cachedPagination
	}
	resp := dtos.PaymentListResponse{
		Payments: payments,
		Meta:     *pagination,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payments fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Payments fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Update payment by ID
// @Summary Update Payment
// @Description Update Payment
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/pyaments/{payment_id} [PATCH]
func UpdatePaymentHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Payments")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.PaymentUpdate](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}
	paymentID := mux.Vars(r)["payment_id"]
	if err := models.UpdatePayment(*req, paymentID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to update payment with ID " + paymentID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("payments_")
	utils.DeleteCacheByPrefix("payments_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentWithID + paymentID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func DeletePaymentHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Payments")
	if !ok {
		return
	}

	paymentID := mux.Vars(r)["payment_id"]
	if err := models.DeletePayment(paymentID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to delete payment with ID " + paymentID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("payments_")
	utils.DeleteCacheByPrefix("payments_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: paymentWithID + paymentID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary  Request refund
// @Description Request refund
// @Tags Payments
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/payments/refund [POST]
func RequestRefund(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "User not found in context",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.Refund](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}
	req.Status = "requested"
	err := models.AddRefundRequest(*req, user.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to request refund from user ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("refunds_")
	utils.DeleteCacheByPrefix("refunds_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund requested successfully For user ID " + user.ID,
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Refund requested successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary  Process refund
// @Description Process refund
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/payments/refund/{refund_id} [patch]
func ProcessRefund(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Payments")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.RefundPayload](r, w, requestSummary, start)
	if !ok {
		return
	}
	refundID := mux.Vars(r)["refund_id"]
	if req.Status != "approved" && req.Status != "rejected" && req.Status != "processed" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Invalid status passed",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid status passed",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	err := models.ProcessRefund(*req, refundID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to process refund with ID " + refundID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund with ID " + refundID + " processed successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Refund status updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary  Get all refunds
// @Description Get all refunds
// @Tags Payments
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/refunds [GET]
func ListRefundsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyRefunds := fmt.Sprintf("refunds_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("refunds_pagination_%d_size_%d", page, size)
	var refunds []dtos.Refund
	var cachedRefunds []dtos.Refund
	var pagination *dtos.PaginationMeta
	var cachedPagination *dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyRefunds, &cachedRefunds)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedRefunds == nil {
		var err error
		refunds, pagination, err = models.ListRefunds(page, size)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: "Failed to list refunds",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKeyRefunds, cachedRefunds)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		refunds = cachedRefunds
		pagination = cachedPagination
	}
	response := dtos.PaginatedRefundsResponse{
		Refunds: refunds,
		Meta:    *pagination,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refunds fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Refunds fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary  Get refund by ID
// @Description Get refund by ID
// @Tags Payments
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/refunds/{refund_id} [GET]
func GetRefundByIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	idStr := mux.Vars(r)["refund_id"]
	id, _ := strconv.Atoi(idStr)
	refund, err := models.GetRefundByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get refund by ID " + idStr,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund with ID " + idStr + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   refund,
		Message:   "Refund fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary  Get user by ID
// @Description Get user by ID
// @Tags Payments
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/refunds/{user_id} [GET]
func GetRefundByUserIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	idStr := mux.Vars(r)["user_id"]
	id, _ := strconv.Atoi(idStr)
	refund, err := models.GetRefundByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get refund by User ID " + idStr,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Refund with User ID " + idStr + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   refund,
		Message:   "Refund fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Create Voucher
// @Description Create Voucher details
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/vouchers [post]
func CreateVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		return
	}
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "User not found in context while creating voucher",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.Voucher](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}
	_, err := models.AddNewVoucher(*req, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Voucher created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func CreatePaymentOptionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Payments")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.PaymentOption](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}
	err := models.CreatePaymentOption(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to create payment option",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment option created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Payment option created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func ListPaymentOptionsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	paymentOptions, pagination, err := models.ListPaymentOptions(q, status, page, size)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to list pay bills",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Pay bills fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"payment_options": paymentOptions, "meta": pagination},
		Message:   "Payment options fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func GetPaymentOptionByIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	paymentOptionID := mux.Vars(r)["payment_option_id"]
	paymentOption, err := models.GetPaymentOptionByID(paymentOptionID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to get payment option by ID " + paymentOptionID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment option with ID " + paymentOptionID + " got successfully",
			Code:        http.StatusOK,
		},
		Payload:   paymentOption,
		Message:   "Payment option fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func UpdatePaymentOptionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Payments")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.PaymentOptionUpdate](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}
	paymentOptionID := mux.Vars(r)["payment_option_id"]
	if err := models.UpdatePaymentOption(paymentOptionID, *req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to update payment option with ID " + paymentOptionID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "payment option With ID" + paymentOptionID + " was updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment option updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func DeletePaymentOptionHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Payments")
	if !ok {
		return
	}

	paymentOptionID := mux.Vars(r)["payment_option_id"]
	if err := models.DeletePaymentOption(paymentOptionID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Failed to delete payment option with ID " + paymentOptionID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "payment option With ID" + paymentOptionID + " was deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Payment option deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
