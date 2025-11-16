package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/payments"
	"adenzo_backend/utils"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

var voucherWithID = "Voucher with ID "

func CreateVoucherDesign(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers"); !ok {
		return
	}

	// Parse multipart form (20 MB max)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to parse form data when creating voucher design: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Get image file
	file, header, err := r.FormFile("image")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to get image file from form data when creating voucher design: " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   "Image is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	defer file.Close()

	// Upload image to GCS
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to upload image to GCS when creating voucher design: " + err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   uploadImageError,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	name := r.FormValue("name")
	status := r.FormValue("status")
	req := dtos.VoucherDesign{
		URL:    url,
		Name:   &name,
		Status: &status,
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}
	// Insert category into DB
	_, err = models.CreateVoucherDesign(url, name, status)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher design in DB: " + err.Error(),
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

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher design added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   url,
		Message:   "Voucher design added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// @Summary Create Voucher
// @Description Create Voucher details
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/vouchers [post]
func CreateVoucherHandlerTest(w http.ResponseWriter, r *http.Request) {
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
				Description: "User not validated/unauthorized",
				Code:        http.StatusUnauthorized,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.VoucherDataCreate](r, w, requestSummary, start)
	if !ok {
		return
	}
	if req.IsToExpire {
		//set expiry date to 90 days from now if is_to_expire is true
		req.ExpiryDate = time.Now().Add(90 * 24 * time.Hour).Format("2006-01-02")
	} else {
		//set expiry date to 30 years from now if is_to_expire is false
		req.ExpiryDate = time.Now().Add(30 * 365 * 24 * time.Hour).Format("2006-01-02")
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}
	voucherID, err := models.CreateNewVoucher(*req, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	data := dtos.BuyVoucherData{
		Amount:       req.Amount,
		FromName:     req.FromName,
		ToName:       req.ToName,
		ToEmail:      req.ToEmail,
		Message:      req.Message,
		DeliveryTime: req.DeliveryTime,
	}
	err = models.InsertIntoVoucherPurchases(data, authuser.ID, voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to record voucher purchase",
				Code:        http.StatusBadRequest,
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
			Description: "Voucher added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   voucherID,
		Message:   "Voucher created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary List Voucher
// @Description List Voucher
// @Tags Vouchers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/vouchers [get]
func ListVouchersHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		return
	}
	isRedeemed := r.URL.Query().Get("is_redeemed")
	status := r.URL.Query().Get("status")
	r.URL.Query().Get("page")
	code := r.URL.Query().Get("code")
	customer := r.URL.Query().Get("customer")
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	vouchers, pagination, err := models.ListVouchers(page, size, isRedeemed, status, code, customer)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers list: " + err.Error(),
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
			Module:      "Vouchers",
			Description: "All vouchers fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"vouchers":   vouchers,
			"pagination": pagination,
		},
		Message:   "Vouchers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// @Summary List Voucher by ID
// @Description List Voucher details
// @Tags Vouchers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/vouchers/{voucher_id} [get]
func GetVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		return
	}
	voucherID := mux.Vars(r)["voucher_id"]

	voucher, err := models.GetVoucherByID(voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher with ID " + voucherID,
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
			Module:      "Vouchers",
			Description: voucherWithID + voucherID + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   voucher,
		Message:   "Voucher fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Update Voucher by ID
// @Description Update Voucher details
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/vouchers/{voucher_id} [delete]
func DeleteVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		return
	}
	voucherID := mux.Vars(r)["voucher_id"]

	err := models.DeleteVoucher(voucherID)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to delete voucher",
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
			Description: "Voucher deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Update Voucher by ID
// @Description Update Voucher details
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/vouchers/{voucher_id} [patch]
func UpdateVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		return
	}
	voucherID := mux.Vars(r)["voucher_id"]
	req, ok := DecodeRequestBody[dtos.VoucherDataUpdate](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}
	err := models.VoucherUpdate(*req, voucherID)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher with ID " + voucherID,
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
			Description: voucherWithID + voucherID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary List user Voucher by ID
// @Description List user Voucher details
// @Tags Vouchers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/vouchers/{voucher_id} [get]
func GetUserVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	voucherID := mux.Vars(r)["voucher_id"]

	voucher, err := models.GetUserVoucherByID(voucherID, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher with ID " + voucherID + " for user with ID " + fmt.Sprint(authuser.ID),
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
			Module:      "Vouchers",
			Description: voucherWithID + voucherID + " fetched successfully for user with ID " + fmt.Sprint(authuser.ID),
			Code:        http.StatusOK,
		},
		Payload:   voucher,
		Message:   "Voucher fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary List user Vouchers
// @Description List user Voucher details
// @Tags Vouchers
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/vouchers/{voucher_id} [get]
func ListUserVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	voucher, pagination, err := models.GetUserVouchers(authuser.ID, page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers for user with ID " + fmt.Sprint(authuser.ID),
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
			Module:      "Vouchers",
			Description: "Vouchers fetched successfully for user with ID " + fmt.Sprint(authuser.ID),
			Code:        http.StatusOK,
		},
		Payload:   map[string]interface{}{"vouchers": voucher, "pagination": pagination},
		Message:   "Vouchers fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

//Endpoint for users to buy vouchers

// @Summary Buy Voucher
// @Description Buy Voucher
// @Tags User
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/vouchers [post]
func BuyVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: noUserFound,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.BuyVoucherData](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}
	// create voucher data
	var voucher dtos.Voucher
	voucher.Amount = req.Amount
	voucher.DesignID = &req.DesignID
	active := "scheduled"
	voucher.Status = &active

	deliveryTime := models.StringToTime(req.DeliveryTime)
	now := time.Now()

	// If delivery time is in the past, mark as active
	if deliveryTime.Before(now) {
		active = "active"
		voucher.Status = &active
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

	voucherID, err := models.AddNewVoucher(voucher, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to create voucher",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//insert into voucher purchases
	err = models.InsertIntoVoucherPurchases(*req, authuser.ID, voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to record voucher purchase",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//create voucher order
	voucherOrderID, err := models.CreateVoucherOrder(req.Amount, voucherID, req.PaymentMethod)
	err = voucherPaymentProcessor(req.PaymentMethod, voucherOrderID, req.PhoneNumber, req.Amount)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to process voucher payment",
				Code:        http.StatusBadRequest,
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
			Description: "Voucher created and added successfully",
			Code:        http.StatusCreated,
		},
		Payload: map[string]interface{}{
			"voucher_order_id": voucherOrderID,
		},
		Message:   "Voucher created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func BuyVoucherUpdateHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	// authuser, ok := middleware.UserFromContext(r.Context())
	// if !ok {
	// 	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
	// 		CollectiveInfo: utils.CollectiveInfo{
	// 			Module:      "Vouchers",
	// 			Description: "User not validated or unauthorized",
	// 			Code:        http.StatusInternalServerError,
	// 		},
	// 		Message:   noUser,
	// 		TimeTaken: time.Since(start),
	// 		Function:  utils.GetCurrentFuncName(),
	// 		Request:   r,
	// 		RawBody:   requestSummary})
	// 	return
	// }
	req, ok := DecodeRequestBody[dtos.BuyVoucherData](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}
	voucherID := mux.Vars(r)["voucher_id"]
	// create voucher data
	amount := req.Amount
	status := "scheduled"

	deliveryTime := models.StringToTime(req.DeliveryTime)
	now := time.Now()

	// If delivery time is in the past, mark as active
	if deliveryTime.Before(now) {
		status = "active"
	}
	if req.Amount > 0 {
		err := models.UpdateVoucher(voucherID, amount, status, req.DesignID)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: "Failed to update voucher",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	//insert into voucher purchases
	err := models.UpdateVoucherPurchases(*req, voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher purchase",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//create voucher order
	voucherOrderID, err := models.UpdateVoucherPurchaseAmount(req.Amount, voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher purchase amount",
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if req.Amount > 0 || req.PaymentMethod != "none" {
		err = voucherPaymentProcessor(req.PaymentMethod, voucherOrderID, req.PhoneNumber, req.Amount)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Vouchers",
					Description: "Failed to process voucher payment",
					Code:        http.StatusBadRequest,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
	}
	utils.DeleteCacheByPrefix("vouchers_")
	utils.DeleteCacheByPrefix("vouchers_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher updated and added successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"voucher_order_id": voucherOrderID,
		},
		Message:   "Voucher updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func voucherPaymentProcessor(paymentMethod string, voucherOrderID, phoneNumber string, amount float64) error {
	switch paymentMethod {
	case "mpesa":
		// Initiate Mpesa payment
		err := payments.HandleMpesaVoucherPayment(voucherOrderID, phoneNumber, amount)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported payment method: %s", paymentMethod)
	}
	return nil
}

// Handler to make voucher mine(redeem to my account)
func RedeemVoucherHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "User not validated or unauthorized",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	req, ok := DecodeRequestBody[dtos.RedeemVoucherRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Vouchers") {
		return
	}
	_, err := models.RedeemVoucher(req.Code, authuser.ID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to redeem voucher",
				Code:        http.StatusBadRequest,
			},
			Message: err.Error(),

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
			Description: "Voucher redeemed successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher redeemed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Sheduler to send bought for voucher emails
func StartVoucherEmailScheduler(interval time.Duration, repeat int) {
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for range ticker.C {
			SendBoughtForVoucherEmails()

			if repeat > 0 {
				repeat--
				if repeat == 0 {
					return
				}
			}
		}
	}()
}

func SendBoughtForVoucherEmails() {
	users, err := models.GetUsersWithUnsentVoucherEmails()
	if err != nil {
		fmt.Printf("error fetching users with unsent voucher emails: %v", err)
		return
	}
	for _, u := range users {
		// notify.SendEmail
		if u.ToEmail != "" {
			subject, htmlBody := utils.SendVoucherEmail(u)
			notification.SendEmail(u.ToEmail, subject, htmlBody)

			//update email sent status
			err := models.MarkVoucherEmailAsSent(u.VoucherID)
			if err != nil {
				log.Printf("error marking voucher email as sent for user: %v", err)
			}
		}
	}
}

func EditVoucherDesign(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers"); !ok {
		return
	}
	designID := mux.Vars(r)["design_id"]

	// Parse multipart form (20 MB max)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}

	// Get image file
	file, header, err := r.FormFile("image")
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Image is required and was not provided",
				Code:        http.StatusBadRequest,
			},
			Message:   "Image is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	defer file.Close()

	// Upload image to GCS
	url, err := utils.UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: err.Error(),
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
		})
		return
	}
	// Insert category into DB
	err = models.EditVoucherDesign(designID, url, r.FormValue("name"), r.FormValue("status"))
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to update voucher design",
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

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher design updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   url,
		Message:   "Voucher design updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func DeleteVoucherDesign(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers"); !ok {
		return
	}
	designID := mux.Vars(r)["voucher_id"]
	// Insert category into DB
	err := models.DeleteVoucherDesign(designID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to delete voucher design",
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

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher design deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Voucher design deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
func GetAllVoucherDesigns(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	name := r.URL.Query().Get("name")
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	designs, pagination, err := models.GetAllVoucherDesigns(page, size, name)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch voucher designs",
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

	// Respond success
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Vouchers",
			Description: "Voucher designs fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   map[string]any{"designs": designs, "pagination": pagination},
		Message:   "Voucher designs fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func ListVoucherPurchasesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		return
	}
	customerName := r.URL.Query().Get("name")
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	vouchers, pagination, err := models.ListVoucherPurchases(page, size, customerName)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers",
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
			Module:      "Vouchers",
			Description: "Vouchers purchases fetched successfully for all users",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"vouchers":   vouchers,
			"pagination": pagination,
		},
		Message:   "Vouchers purchases fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}
func GetVoucherPurchasesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Vouchers")
	if !ok {
		return
	}
	voucherID := mux.Vars(r)["purchase_id"]
	vouchers, err := models.GetVoucherPurchases(voucherID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Vouchers",
				Description: "Failed to fetch vouchers",
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
			Module:      "Vouchers",
			Description: "Vouchers purchase fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   vouchers,
		Message:   "Vouchers purchase fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}
