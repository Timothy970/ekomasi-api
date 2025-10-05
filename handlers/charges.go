package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Add Charge
func AddChargeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.Charge](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	charge, err := models.AddCharge(*req)
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK, Payload: charge, Message: "Charge added successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary})
}

// Update Charge
func UpdateChargeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	id := mux.Vars(r)["charge_id"]

	req, ok := DecodeRequestBody[dtos.Charge](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	charge, err := models.UpdateCharge(id, *req)
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
		Code: http.StatusOK, Payload: charge, Message: "Charge updated successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary,
	})
}

// Get Charge by ID
func GetChargeByIDHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	// if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
	// 	return
	// }

	id := mux.Vars(r)["charge_id"]
	charge, err := models.GetChargeByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{Code: http.StatusInternalServerError, Message: err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{Code: http.StatusOK, Payload: charge, Message: "Charge retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get All Charges
func GetAllChargesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	// if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
	// 	return
	// }

	charges, err := models.GetAllCharges()
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{Code: http.StatusInternalServerError, Message: err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{Code: http.StatusOK, Payload: charges, Message: "Charges retrieved successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary})
}

// Delete Charge
func DeleteChargeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}

	id := mux.Vars(r)["charge_id"]
	err := models.DeleteCharge(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{Code: http.StatusInternalServerError, Message: err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{Code: http.StatusOK, Message: "Charge deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Add charge to product
func AddChargeToProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.AddChargeToProductRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.AddChargeToProduct(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{Code: http.StatusInternalServerError, Message: err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{Code: http.StatusOK, Message: "Charge added to product successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
