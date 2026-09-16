// Package handlers provides HTTP request handlers for shipping and delivery management.
// This file contains handlers for shipping cost calculation, location management, delivery feedback,
// and related logistics operations. Supports shipping rate configuration, location-based pricing,
// and customer feedback collection for delivery quality assessment.
package handlers

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

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
func GetUserDeliveryFeedbacks(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract user ID from URL path parameters
	userID := c.Param("user_id")
	// Fetch all feedback submitted by the user from database
	feedback, err := models.GetDeliveryUserFeedBack(models.DB, userID)
	if err != nil {
		// Handle different error types with appropriate responses
		if err == sql.ErrNoRows {
			// User has not submitted any feedback yet
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Delivery feedback not found for user ID " + userID,
					Code:        http.StatusNotFound,
				},
				Message:   "Delivery feedback not found",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
		} else {
			// Database query failed
			log.Printf("error getting feedback:::%v", err)
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "Error fetching feedback for user ID " + userID,
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
		}
		return
	}

	// Return user's feedback history for satisfaction tracking

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Feedback fetched successfully for user ID " + userID,
			Code:        http.StatusOK,
		},
		Payload:   feedback,
		Message:   "Feedback",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DecodeRequestBody is a generic helper function for parsing JSON request bodies using Gin context.
// Provides consistent error handling and validation across all handlers.
// Detects malformed JSON, type mismatches, and unknown fields automatically.
func DecodeRequestBody[T any](c *gin.Context, requestSummary string, start time.Time) (*T, bool) {
	var req T
	// Initialize JSON decoder with strict validation
	decoder := json.NewDecoder(c.Request.Body)
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
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Requests",
				Description: msg,
				Code:        http.StatusBadRequest,
			},
			Message:   msg,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
// @Success      200   {object}  map[string]any    "Locations with pagination"
// @Failure      400   {object}  dtos.ErrorResponse        "Failed to retrieve locations"
// @Router       /api/locations [get]
func ListLocations(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Parse pagination parameters from query string
	page, size := parsePagination(c.Query("page"), c.Query("size"))
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
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Shipping",
					Description: "Failed to retrieve locations",
					Code:        http.StatusBadRequest,
				},
				Message:   fmt.Sprintf("%s", err),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
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
	response := map[string]any{
		"locations":  locations,
		"pagination": pagination,
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Shipping",
			Description: "Locations fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Locations fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func GetLocation(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract location ID from URL path parameters
	locationID := c.Param("location_id")
	// Convert location ID string to integer
	id, _ := strconv.Atoi(locationID)
	// Fetch location details from database
	loc, err := models.GetLocationByID(models.DB, id)
	if err != nil {
		// Location not found or database error
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to fetch location details",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Return location details for display
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Shipping",
			Description: "Location fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   loc,
		Message:   "Location fetched sucessfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
