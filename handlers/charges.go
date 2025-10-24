package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var chargeWithID = "Charge with ID "

// Add Charge
func AddChargeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Charges"); !ok {
		return
	}

	req, ok := DecodeRequestBody[dtos.Charge](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Charges") {
		return
	}

	charge, err := models.AddCharge(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Charges",
				Description: "Failed to add charge",
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
			Module:      "Charges",
			Description: "Charge added successfully",
			Code:        http.StatusCreated,
		},
		Payload: charge, Message: "Charge added successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary})
}

// Update Charge
func UpdateChargeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Charges"); !ok {
		return
	}

	id := mux.Vars(r)["charge_id"]

	req, ok := DecodeRequestBody[dtos.Charge](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Charges") {
		return
	}
	charge, err := models.UpdateCharge(id, *req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Charges",
				Description: "Failed to update charge with ID " + id,
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
			Module:      "Charges",
			Description: chargeWithID + id + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload: charge, Message: "Charge updated successfully", TimeTaken: time.Since(start),
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
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Charges",
			Description: "Failed to get charge with ID " + id,
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
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
		Module:      "Charges",
		Description: chargeWithID + id + " retrieved successfully",
		Code:        http.StatusOK,
	},
		Payload: charge, Message: "Charge retrieved successfully",
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
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Charges",
			Description: "Failed to get all charges",
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
		Module:      "Charges",
		Description: "All charges retrieved successfully",
		Code:        http.StatusOK,
	},
		Payload: charges, Message: "Charges retrieved successfully", TimeTaken: time.Since(start),
		Function: utils.GetCurrentFuncName(),
		Request:  r,
		RawBody:  requestSummary})
}

// Delete Charge
func DeleteChargeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Charges"); !ok {
		return
	}

	id := mux.Vars(r)["charge_id"]
	err := models.DeleteCharge(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Charges",
			Description: "Failed to delete charge with ID " + id,
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
		Module:      "Charges",
		Description: chargeWithID + id + " deleted successfully",
		Code:        http.StatusOK,
	},
		Payload:   nil,
		Message:   "Charge deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Add charge to product
func AddChargeToProductHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary, "Charges"); !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.AddChargeToProductRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Charges") {
		return
	}
	err := models.AddChargeToProduct(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{CollectiveInfo: utils.CollectiveInfo{
			Module:      "Charges",
			Description: "Failed to add charge to product",
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
		Module:      "Charges",
		Description: "Charge with charge ID " + req.ChargeID + " added to product with product ID " + req.ProductID + " successfully",
		Code:        http.StatusOK,
	},
		Payload:   nil,
		Message:   "Charge added to product successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
