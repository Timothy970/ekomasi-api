package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func GetShippingCostHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.ShippingCostRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	// location := strings.ToLower(strings.TrimSpace(req.Location))
	// cacheKey := fmt.Sprintf("delivery_rate:%s", location)

	// // Try Redis cache first
	// cached, err := Redis.Get(context.Background(), cacheKey).Result()
	// if err == nil {
	// 	json.NewEncoder(w).Encode(dtos.ShippingCostResponse{
	// 		Location: req.Location,
	// 		Charge:   cached,
	// 	})
	// 	return
	// }

	charge, dbResult, err := models.GetDeliveryRate(req.Location)

	if err != nil {
		log.Printf("Server error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Failed to fetch delivery rates",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Cache result in Redis
	// _ = Redis.Set(context.Background(), cacheKey, charge, 24*time.Hour).Err()
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
		Payload: dtos.ShippingCostResponse{
			Location: dbResult,
			Charge:   charge,
		},
		Message:   "Promotions fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// @Summary Craete Location
// @Description Create Location
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/locations [post]
func StoreShippingRates(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.ShippingCostResponse](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	err := models.AddNewShippingRate(*req)
	if err != nil {
		log.Printf("Error adding new shipping rate: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("locations_")
	utils.DeleteCacheByPrefix("locations_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Location added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// submit delivery feed back
// @Summary Submit Delivery feedback
// @Description Submit delivery feedback
// @Tags Delivery Feedback
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/deliveries/feedback [post]
func SubmitFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.DeliveryFeedback](r, w, requestSummary, start)
	if !ok {
		return
	}
	err := models.AddNewDeliveryFeedback(*req)
	if err != nil {
		log.Printf("Error adding new feed back: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Feedback submitted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get delivery feedback
// @Summary Get Delivery feedback
// @Description Get delivery feedback
// @Tags Delivery Feedback
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/feedbacks [get]
func GetDeliveryFeedbacks(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// ctx := r.Context()
	deliveryID := mux.Vars(r)["delivery_id"]

	feedback, err := models.GetDeliveryFeedBack(deliveryID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   "No delivery feedback found",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			log.Printf("error getting feedback:::%v", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   "Error fetching feedback",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		}
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   feedback,
		Message:   "Feedback",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get delivery feedback
// @Summary Get User Delivery feedback
// @Description Get user delivery feedback
// @Tags Delivery Feedback
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/deliveries/feedback/{user_id} [get]
func GetUserDeliveryFeedbacks(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// ctx := r.Context()
	userID := mux.Vars(r)["user_id"]
	feedback, err := models.GetDeliveryUserFeedBack(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusNotFound,
				Message:   "Delivery feedback not found",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			log.Printf("error getting feedback:::%v", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		}
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   feedback,
		Message:   "Feedback",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func DecodeRequestBody[T any](r *http.Request, w http.ResponseWriter, requestSummary string, start time.Time) (*T, bool) {
	var req T
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // helps catch unexpected fields

	if err := decoder.Decode(&req); err != nil {
		var msg string
		log.Printf("Error decoding request body: %v", err)
		switch e := err.(type) {
		case *json.SyntaxError:
			msg = fmt.Sprintf("Request body contains badly-formed data (at position %d)", e.Offset)
		case *json.UnmarshalTypeError:
			msg = fmt.Sprintf("Request body has invalid type for field %q at position %d. Expected %v",
				e.Field, e.Offset, e.Type)
		default:
			msg = "Invalid request body: " + err.Error()
		}

		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   msg,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return nil, false
	}

	return &req, true
}

// List Locations (with pagination)
//
// @Summary List Location
// @Description List Location
// @Tags Locations
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/locations [get]
func ListLocations(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyLocations := fmt.Sprintf("locations_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("locations_pagination_%d_size_%d", page, size)
	var locations []dtos.Location
	var cachedLocation []dtos.Location
	var pagination dtos.PaginationMeta
	var cachedPagination dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyLocations, &cachedLocation)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedLocation == nil {
		var err error
		locations, pagination, err = models.ListLocations(page, size)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusBadRequest,
				Message:   fmt.Sprintf("%s", err),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return

		}
		_ = utils.SetCache(cacheKeyLocations, cachedLocation)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		locations = cachedLocation
		pagination = cachedPagination
	}
	response := map[string]interface{}{
		"locations":  locations,
		"pagination": pagination,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   response,
		Message:   "Locations fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Get Location details
//
// @Summary List Location details
// @Description List Location details
// @Tags Locations
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/locations/{location_id} [get]
func GetLocation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	locationID := mux.Vars(r)["location_id"]
	id, _ := strconv.Atoi(locationID)
	loc, err := models.GetLocationByID(id)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   loc,
		Message:   "Location fetched sucessfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Update Location
//
// @Summary Update Location
// @Description Update Location details
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/locations/{location_id} [patch]
func UpdateLocation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	locationID := mux.Vars(r)["location_id"]
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateLocation](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	if err := models.UpdateLocation(*req, locationID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("locations_")
	utils.DeleteCacheByPrefix("locations_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Location updated sucessfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete Location
//
// @Summary Delete Location
// @Description Delete Location details
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/admin/locations/{location_id} [delete]
func DeleteLocation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	locationID := mux.Vars(r)["location_id"]
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}

	if err := models.DeleteLocation(locationID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("locations_")
	utils.DeleteCacheByPrefix("locations_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Location deleted sucessfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// delete delivery feed back
// @Summary Delete Delivery feedback
// @Description Delete delivery feedback
// @Tags Delivery Feedback
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/deliveries/feedback/{feedback_id} [delete]
func DeleteFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	feedbackID := mux.Vars(r)["feedback_id"]

	err := models.DeleteDeliveryFeedback(feedbackID)
	if err != nil {
		log.Printf("Error adding new feed back: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusBadRequest,
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Feedback deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
