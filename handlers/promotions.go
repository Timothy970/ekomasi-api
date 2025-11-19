package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var promoCodeWithID = "Promo code with ID "

// Add PromoCode
func AddPromoCodeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Promotions"); !ok {
		return
	}
	minimumOrderValue := models.StringToFloat64(r.FormValue("minimum_order_value"))

	req := &dtos.PromoCodeRequest{
		Discount_Code:     r.FormValue("discount_code"),
		DiscountType:      r.FormValue("discount_type"),
		DiscountValue:     models.StringToFloat64(r.FormValue("discount_value")),
		ExpiresAt:         r.FormValue("expires_at"),
		MinimumOrderValue: &minimumOrderValue,
		MaximumUse:        int(models.StringToFloat64(r.FormValue("maximum_use"))),
		IsActive:          models.StringToBool(r.FormValue("is_active")),
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Promotions") {
		return
	}

	promo, err := models.AddPromoCode(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to add promo code",
				Code:        http.StatusInternalServerError,
			},
			Message: err.Error(), TimeTaken: time.Since(start),
			Function: utils.GetCurrentFuncName(),
			Request:  r,
			RawBody:  requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Promo code added successfully",
			Code:        http.StatusOK,
		},
		Payload: promo, Message: "Promo code added successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}

// Update PromoCode
func UpdatePromoCodeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Promotions"); !ok {
		return
	}

	id := mux.Vars(r)["promo_id"]
	minimumOrderValue := models.StringToFloat64(r.FormValue("minimum_order_value"))
	req := &dtos.PromoCodeRequest{
		Discount_Code:     r.FormValue("discount_code"),
		DiscountType:      r.FormValue("discount_type"),
		DiscountValue:     models.StringToFloat64(r.FormValue("discount_value")),
		ExpiresAt:         r.FormValue("expires_at"),
		MinimumOrderValue: &minimumOrderValue,
		MaximumUse:        int(models.StringToFloat64(r.FormValue("maximum_use"))),
		IsActive:          models.StringToBool(r.FormValue("is_active")),
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Promotions") {
		return
	}
	promo, err := models.UpdatePromoCode(id, *req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to update promo code with ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message: err.Error(), TimeTaken: time.Since(start),
			Function: utils.GetCurrentFuncName(),
			Request:  r,
			RawBody:  requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: promoCodeWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload: promo, Message: "Promo code updated successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}

// Get PromoCode by ID
func GetPromoCodeByIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Promotions"); !ok {
		return
	}
	id := mux.Vars(r)["promo_id"]
	promo, err := models.GetPromoCodeByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Failed to get promo code with ID " + id,
			Code:        http.StatusNotFound,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Promotions",
		Description: promoCodeWithID + id + " retrieved successfully",
		Code:        http.StatusOK,
	},
		Payload: promo, Message: "Promo code retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get All PromoCodes
func GetAllPromoCodesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Promotions"); !ok {
		return
	}
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	promos, pagination, err := models.GetAllPromoCodes(page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Failed to retrieve all promo codes",
			Code:        http.StatusInternalServerError,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Promotions",
		Description: "All promo codes retrieved successfully",
		Code:        http.StatusOK,
	},
		Payload: map[string]any{"promocodes": promos, "pagination": pagination}, Message: "Promo codes retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete PromoCode
func DeletePromoCodeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Promotions"); !ok {
		return
	}

	id := mux.Vars(r)["promo_id"]

	err := models.DeletePromoCode(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Failed to delete promo code with ID " + id,
			Code:        http.StatusInternalServerError,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Promotions",
		Description: promoCodeWithID + id + " deleted successfully",
		Code:        http.StatusOK,
	},
		Payload:   nil,
		Message:   "Promo code deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func TogglePromoCodeStatusHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Promotions"); !ok {
		return
	}

	id := mux.Vars(r)["promo_id"]
	req, ok := DecodeRequestBody[dtos.PromoCodeStatusRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Promotions") {
		return
	}

	err := models.SetPromoCodeActiveStatus(id, req.IsActive)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Promotions",
				Description: "Failed to set promo code active status for ID " + id,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	msg := "Promo code deactivated successfully"
	if req.IsActive {
		msg = "Promo code activated successfully"
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: msg + " for ID " + id,
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   msg,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Add a product to a promotion type
func AddPromotionToProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Promotions"); !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.AddPromotionToProductRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Promotions") {
		return
	}
	if err := models.AddPromotionToProduct(*req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Promotions",
			Description: "Failed to add promotion to product with ID " + req.ProductID,
			Code:        http.StatusInternalServerError,
		},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Promotions",
		Description: "Promotion added to product with ID " + req.ProductID + " successfully",
		Code:        http.StatusOK,
	},
		Payload:   nil,
		Message:   "Promotion added to product successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
