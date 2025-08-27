package payments

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func HandleVoucherPayment(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	var req dtos.VoucherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   "Invalid request",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	// check validity of voucher code
	voucher, ok := validateVoucher(w, r, req, start, requestSummary)
	if !ok {
		return
	}
	// set voucher as redeemed
	if err := models.SetVoucherAsRedeemed(voucher.Code); err != nil {
		respondWithVoucherError(w, r, start, requestSummary, "Failed to redeem voucher")

	}
	err := models.UpdateDeliveryOrderTables(req.DeliveryID, req.OrderID)
	if err != nil {
		log.Printf("%v", err)
		respondWithVoucherError(w, r, start, requestSummary, "Failed to make payment")

	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Payment done successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
func validateVoucher(w http.ResponseWriter, r *http.Request, req dtos.VoucherRequest, start time.Time, requestSummary string) (*dtos.Voucher, bool) {
	voucher, err := models.GetVoucherByCode(req.VoucherCode)
	if err != nil {
		respondWithVoucherError(w, r, start, requestSummary, "Failed to retrieve voucher")
		return nil, false
	}
	if voucher == nil {
		respondWithVoucherError(w, r, start, requestSummary, "Voucher not found")
		return nil, false
	}
	if voucher.IsRedeemed {
		respondWithVoucherError(w, r, start, requestSummary, "Voucher has already been redeemed")
		return nil, false
	}
	if voucher.Amount < float64(req.Amount) {
		respondWithVoucherError(w, r, start, requestSummary, "Voucher amount is not sufficient for this payment")
		return nil, false
	}
	return voucher, true
}
func respondWithVoucherError(w http.ResponseWriter, r *http.Request, start time.Time, rawBody, message string) {
	utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
		Code:      http.StatusBadRequest,
		Message:   message,
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   rawBody,
	})
}
