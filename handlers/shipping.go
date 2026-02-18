// Package handlers provides HTTP request handlers for shipping and delivery management.
// This file contains handlers for shipping cost calculation, location management, delivery feedback,
// and related logistics operations. Supports shipping rate configuration, location-based pricing,
// and customer feedback collection for delivery quality assessment.
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

// GetShippingCostHandler calculates shipping costs based on delivery location.
// Provides real-time shipping cost estimates for checkout and order processing.
// Essential for transparent pricing and accurate total cost calculation.
//
// @Summary      Calculate shipping cost
// @Description  Get shipping cost estimate for a specific delivery location
// @Tags         Shipping
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.ShippingCostRequest   true  "Shipping location details"
// @Success      200      {object}  dtos.ShippingCostResponse  "Shipping cost calculated"
// @Failure      500      {object}  dtos.ErrorResponse         "Failed to calculate cost"
// @Router       /api/shipping/cost [post]
func GetShippingCostHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Decode and parse JSON request body with location details
	req, ok := DecodeRequestBody[dtos.ShippingCostRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// TODO1: Implement Redis caching for delivery rates to improve performance
	// location := strings.ToLower(strings.TrimSpace(req.Location))
	// cacheKey := fmt.Sprintf("delivery_rate:%s", location)
	// Try Redis cache first before database query
	// cached, err := Redis.Get(context.Background(), cacheKey).Result()
	// if err == nil {
	// 	return cached rate from Redis
	// }

	// Query database for delivery rate based on location
	charge, dbResult, err := models.GetDeliveryRate(models.DB, req.Location)

	if err != nil {
		// Database query failed or location not found
		log.Printf("Server error::%s", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to fetch delivery rates",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// TODO1: Cache delivery rate result in Redis for 24 hours
	// _ = Redis.Set(context.Background(), cacheKey, charge, 24*time.Hour).Err()
	// Return shipping cost response with location and charge details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Shipping",
			Description: "Delivery rates fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: dtos.ShippingCostResponse{
			Location: dbResult,
			Charge:   charge,
		},
		Message:   "Delivery rates fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})

}

// StoreShippingRates adds a new delivery location with associated shipping cost.
// Admin-only operation for configuring location-based shipping rates in the system.
// Essential for expanding delivery coverage and managing regional pricing.
//
// @Summary      Create shipping location
// @Description  Add a new delivery location with shipping rate (admin only)
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        location  body      dtos.ShippingCostResponse  true  "Location and rate details"
// @Success      200       {object}  map[string]interface{}       "Location added successfully"
// @Failure      400       {object}  dtos.ErrorResponse         "Invalid request or duplicate location"
// @Failure      401       {object}  dtos.ErrorResponse         "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/locations [post]
func StoreShippingRates(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can configure shipping rates)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Shipping", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.ShippingCostResponse](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Shipping") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Create new shipping rate record in database
	err := models.AddNewShippingRate(models.DB, *req)
	if err != nil {
		// Shipping rate creation failed (duplicate location or database error)
		log.Printf("Error adding new shipping rate: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to add new shipping rate",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate location caches to ensure fresh data
	utils.DeleteCacheByPrefix("locations_")
	utils.DeleteCacheByPrefix("locations_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Shipping",
			Description: "Location added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Location added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// SubmitFeedbackHandler allows customers to submit feedback about their delivery experience.
// Collects ratings, comments, and issues to improve delivery service quality.
// Important for customer satisfaction monitoring and delivery partner performance evaluation.
//
// @Summary      Submit delivery feedback
// @Description  Submit customer feedback and rating for a delivery experience
// @Tags         Delivery Feedback
// @Accept       json
// @Produce      json
// @Param        feedback  body      dtos.DeliveryFeedback   true  "Delivery feedback details"
// @Success      200       {object}  map[string]interface{}    "Feedback submitted successfully"
// @Failure      400       {object}  dtos.ErrorResponse      "Invalid feedback data"
// @Router       /api/deliveries/feedback [post]
func SubmitFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Decode and parse JSON request body with feedback details
	req, ok := DecodeRequestBody[dtos.DeliveryFeedback](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Save delivery feedback to database for quality tracking
	err := models.AddNewDeliveryFeedback(models.DB, *req)
	if err != nil {
		// Feedback submission failed (invalid delivery ID or database error)
		log.Printf("Error adding new feed back: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to submit delivery feedback",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Feedback submitted successfully for service improvement
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Feedback submitted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Feedback submitted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetDeliveryFeedbacks retrieves feedback for a specific delivery.
// Allows viewing customer feedback and ratings to assess delivery quality.
// Important for delivery service evaluation and issue resolution.
//
// @Summary      Get delivery feedback
// @Description  Retrieve feedback and rating for a specific delivery
// @Tags         Delivery Feedback
// @Produce      json
// @Param        delivery_id  path      string                  true  "Delivery ID"
// @Success      200          {object}  dtos.DeliveryFeedback   "Feedback details"
// @Failure      404          {object}  dtos.ErrorResponse      "Feedback not found"
// @Failure      500          {object}  dtos.ErrorResponse      "Error fetching feedback"
// @Router       /api/deliveries/{delivery_id}/feedback [get]
func GetDeliveryFeedbacks(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract delivery ID from URL path parameters
	deliveryID := mux.Vars(r)["delivery_id"]

	// Fetch feedback data from database for the specified delivery
	feedback, err := models.GetDeliveryFeedBack(models.DB, deliveryID)
	if err != nil {
		// Handle different error types with appropriate responses
		if err == sql.ErrNoRows {
			// No feedback exists for this delivery
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "No delivery feedback found",
					Code:        http.StatusNotFound,
				},
				Message:   "No delivery feedback found",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			// Database query failed
			log.Printf("error getting feedback:::%v", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Error fetching feedback",
					Code:        http.StatusInternalServerError,
				},
				Message:   "Error fetching feedback",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		}
		return
	}

	// Return feedback data for delivery quality assessment

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Feedback fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   feedback,
		Message:   "Feedback",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetUserDeliveryFeedbacks retrieves all delivery feedback submitted by a specific user.
// Shows user's delivery experience history and feedback patterns.
// Useful for customer service and understanding user satisfaction trends.
//
// @Summary      Get user's delivery feedback
// @Description  Retrieve all delivery feedback submitted by a specific user
// @Tags         Delivery Feedback
// @Produce      json
// @Param        user_id  path      string                    true  "User ID"
// @Success      200      {array}   dtos.DeliveryFeedback     "User's feedback list"
// @Failure      404      {object}  dtos.ErrorResponse        "No feedback found for user"
// @Failure      500      {object}  dtos.ErrorResponse        "Error fetching feedback"
// @Router       /api/deliveries/feedback/user/{user_id} [get]
func GetUserDeliveryFeedbacks(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract user ID from URL path parameters
	userID := mux.Vars(r)["user_id"]
	// Fetch all feedback submitted by the user from database
	feedback, err := models.GetDeliveryUserFeedBack(models.DB, userID)
	if err != nil {
		// Handle different error types with appropriate responses
		if err == sql.ErrNoRows {
			// User has not submitted any feedback yet
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Delivery feedback not found for user ID " + userID,
					Code:        http.StatusNotFound,
				},
				Message:   "Delivery feedback not found",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			// Database query failed
			log.Printf("error getting feedback:::%v", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Error fetching feedback for user ID " + userID,
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		}
		return
	}

	// Return user's feedback history for satisfaction tracking

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Feedback fetched successfully for user ID " + userID,
			Code:        http.StatusOK,
		},
		Payload:   feedback,
		Message:   "Feedback",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DecodeRequestBody is a generic helper function for parsing JSON request bodies.
// Provides consistent error handling and validation across all handlers.
// Detects malformed JSON, type mismatches, and unknown fields automatically.
func DecodeRequestBody[T any](r *http.Request, w http.ResponseWriter, requestSummary string, start time.Time) (*T, bool) {
	var req T
	// Initialize JSON decoder with strict validation
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Reject unexpected fields in JSON

	// Attempt to decode JSON into the generic type T
	if err := decoder.Decode(&req); err != nil {
		var msg string
		log.Printf("Error decoding request body: %v", err)
		// Provide specific error messages based on error type
		switch e := err.(type) {
		case *json.SyntaxError:
			// JSON syntax error with position information
			msg = fmt.Sprintf("Request body contains badly-formed data (at position %d)", e.Offset)
		case *json.UnmarshalTypeError:
			// Type mismatch error with field and expected type
			msg = fmt.Sprintf("Request body has invalid type for field %q at position %d. Expected %v",
				e.Field, e.Offset, e.Type)
		default:
			// Generic decoding error
			msg = "Invalid request body: " + err.Error()
		}

		// Send detailed error response to client
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Requests",
				Description: msg,
				Code:        http.StatusBadRequest,
			},
			Message:   msg,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return nil, false
	}

	// Successfully decoded request body
	return &req, true
}

// ListLocations retrieves all delivery locations with pagination.
// Displays available delivery areas with shipping rates for customer reference.
// Includes Redis caching for optimal performance on frequently accessed data.
//
// @Summary      List delivery locations
// @Description  Retrieve paginated list of all delivery locations with shipping rates
// @Tags         Locations
// @Produce      json
// @Param        page  query     int                       false  "Page number (default: 1)"
// @Param        size  query     int                       false  "Page size (default: 10)"
// @Success      200   {object}  map[string]interface{}    "Locations with pagination"
// @Failure      400   {object}  dtos.ErrorResponse        "Failed to retrieve locations"
// @Router       /api/locations [get]
func ListLocations(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Parse pagination parameters from query string
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Generate cache keys for locations and pagination data
	cacheKeyLocations := fmt.Sprintf("locations_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("locations_pagination_%d_size_%d", page, size)
	var locations []dtos.Location
	var cachedLocation []dtos.Location
	var pagination dtos.PaginationMeta
	var cachedPagination dtos.PaginationMeta
	// Try to fetch locations from Redis cache
	_ = utils.GetCache(cacheKeyLocations, &cachedLocation)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedLocation == nil {
		// Cache miss - fetch from database
		var err error
		locations, pagination, err = models.ListLocations(models.DB, page, size)
		if err != nil {
			// Database query failed
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Shipping",
					Description: "Failed to retrieve locations",
					Code:        http.StatusBadRequest,
				},
				Message:   fmt.Sprintf("%s", err),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return

		}
		// Store results in Redis cache for future requests
		_ = utils.SetCache(cacheKeyLocations, cachedLocation)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		// Cache hit - use cached data
		locations = cachedLocation
		pagination = cachedPagination
	}
	// Build response payload with locations and pagination metadata
	response := map[string]interface{}{
		"locations":  locations,
		"pagination": pagination,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Shipping",
			Description: "Locations fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Locations fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetLocation retrieves detailed information for a specific delivery location.
// Displays location name, delivery area, shipping rate, and availability status.
// Used for location-specific shipping information during checkout.
//
// @Summary      Get location details
// @Description  Retrieve detailed information for a specific delivery location
// @Tags         Locations
// @Produce      json
// @Param        location_id  path      int                 true  "Location ID"
// @Success      200          {object}  dtos.Location       "Location details"
// @Failure      400          {object}  dtos.ErrorResponse  "Location not found"
// @Router       /api/locations/{location_id} [get]
func GetLocation(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract location ID from URL path parameters
	locationID := mux.Vars(r)["location_id"]
	// Convert location ID string to integer
	id, _ := strconv.Atoi(locationID)
	// Fetch location details from database
	loc, err := models.GetLocationByID(models.DB, id)
	if err != nil {
		// Location not found or database error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to fetch location details",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return location details for display
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Shipping",
			Description: "Location fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   loc,
		Message:   "Location fetched sucessfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateLocation modifies an existing delivery location's details.
// Admin-only operation for updating location names, areas, and shipping rates.
// Essential for maintaining accurate delivery zones and pricing.
//
// @Summary      Update location
// @Description  Update delivery location details and shipping rate (admin only)
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        location_id  path      int                   true  "Location ID"
// @Param        location     body      dtos.UpdateLocation   true  "Updated location details"
// @Success      200          {object}  map[string]interface{}  "Location updated successfully"
// @Failure      400          {object}  dtos.ErrorResponse    "Update failed"
// @Failure      401          {object}  dtos.ErrorResponse    "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/locations/{location_id} [patch]
func UpdateLocation(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract location ID from URL path parameters
	locationID := mux.Vars(r)["location_id"]
	// Verify user has admin privileges (only admins can update locations)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Shipping", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.UpdateLocation](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Shipping") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Update location details in database
	if err := models.UpdateLocation(models.DB, *req, locationID); err != nil {
		// Update failed (location not found or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to update location with ID " + locationID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate location caches to ensure fresh data
	utils.DeleteCacheByPrefix("locations_")
	utils.DeleteCacheByPrefix("locations_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Shipping",
			Description: "Location with ID " + locationID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Location updated sucessfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteLocation permanently removes a delivery location from the system.
// Admin-only operation for discontinuing service to specific areas.
// Deletion may fail if location is referenced in active orders.
//
// @Summary      Delete location
// @Description  Permanently remove a delivery location (admin only)
// @Tags         Admin
// @Produce      json
// @Param        location_id  path      int                   true  "Location ID"
// @Success      200          {object}  map[string]interface{}  "Location deleted successfully"
// @Failure      400          {object}  dtos.ErrorResponse    "Deletion failed"
// @Failure      401          {object}  dtos.ErrorResponse    "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/locations/{location_id} [delete]
func DeleteLocation(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract location ID from URL path parameters
	locationID := mux.Vars(r)["location_id"]
	// Verify user has admin privileges (only admins can delete locations)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Shipping", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Permanently delete location from database
	if err := models.DeleteLocation(models.DB, locationID); err != nil {
		// Deletion failed (location not found, in use, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to delete location with ID " + locationID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Invalidate location caches to ensure fresh data
	utils.DeleteCacheByPrefix("locations_")
	utils.DeleteCacheByPrefix("locations_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Shipping",
			Description: "Location with ID " + locationID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Location deleted sucessfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteFeedbackHandler permanently removes delivery feedback from the system.
// Allows deletion of inappropriate or mistaken feedback submissions.
// Should be used sparingly to maintain feedback authenticity.
//
// @Summary      Delete delivery feedback
// @Description  Permanently remove delivery feedback
// @Tags         Delivery Feedback
// @Produce      json
// @Param        feedback_id  path      string                true  "Feedback ID"
// @Success      200          {object}  map[string]interface{}  "Feedback deleted successfully"
// @Failure      400          {object}  dtos.ErrorResponse    "Deletion failed"
// @Router       /api/deliveries/feedback/{feedback_id} [delete]
func DeleteFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract feedback ID from URL path parameters
	feedbackID := mux.Vars(r)["feedback_id"]

	// Permanently delete feedback from database
	err := models.DeleteDeliveryFeedback(models.DB, feedbackID)
	if err != nil {
		// Deletion failed (feedback not found or database error)
		log.Printf("Error adding new feed back: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete delivery feedback with ID " + feedbackID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Feedback deleted successfully
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Feedback with ID " + feedbackID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Feedback deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
