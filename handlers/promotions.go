package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Add PromoCode
func AddPromoCodeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.PromoCodeRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	promo, err := models.AddPromoCode(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code: http.StatusInternalServerError, Message: err.Error(), TimeTaken: time.Since(start),
			Function: utils.GetCurrentFuncName(),
			Request:  r,
			RawBody:  requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK, Payload: promo, Message: "Promo code added successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}

// Update PromoCode
func UpdatePromoCodeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	id := mux.Vars(r)["promo_id"]
	req, ok := DecodeRequestBody[dtos.PromoCodeRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	promo, err := models.UpdatePromoCode(id, *req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code: http.StatusInternalServerError, Message: err.Error(), TimeTaken: time.Since(start),
			Function: utils.GetCurrentFuncName(),
			Request:  r,
			RawBody:  requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK, Payload: promo, Message: "Promo code updated successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}

// Get PromoCode by ID
func GetPromoCodeByIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	id := mux.Vars(r)["promo_id"]
	promo, err := models.GetPromoCodeByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{Code: http.StatusInternalServerError, Message: err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{Code: http.StatusOK, Payload: promo, Message: "Promo code retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get All PromoCodes
func GetAllPromoCodesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	promos, err := models.GetAllPromoCodes()
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{Code: http.StatusInternalServerError, Message: err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{Code: http.StatusOK, Payload: promos, Message: "Promo codes retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete PromoCode
func DeletePromoCodeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	id := mux.Vars(r)["promo_id"]

	err := models.DeletePromoCode(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{Code: http.StatusInternalServerError, Message: err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{Code: http.StatusOK, Message: "Promo code deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func TogglePromoCodeStatusHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	id := mux.Vars(r)["promo_id"]
	req, ok := DecodeRequestBody[dtos.PromoCodeStatusRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	err := models.SetPromoCodeActiveStatus(id, req.IsActive)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
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
		Code:      http.StatusOK,
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
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.AddPromotionToProductRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	if err := models.AddPromotionToProduct(*req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{Code: http.StatusInternalServerError, Message: err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{Code: http.StatusOK, Message: "Promotion added to product successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
