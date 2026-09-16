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
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
func GetShippingCostHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Decode and parse JSON request body with location details
	req, ok := DecodeRequestBody[dtos.ShippingCostRequest](c, requestSummary, start)
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
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to fetch delivery rates",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// TODO1: Cache delivery rate result in Redis for 24 hours
	// _ = Redis.Set(context.Background(), cacheKey, charge, 24*time.Hour).Err()
	// Return shipping cost response with location and charge details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
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
		Request:   c.Request,
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
// @Success      200       {object}  map[string]any       "Location added successfully"
// @Failure      400       {object}  dtos.ErrorResponse         "Invalid request or duplicate location"
// @Failure      401       {object}  dtos.ErrorResponse         "Admin authorization required"
// @Security     BearerAuth
// @Router       /api/admin/locations [post]
func StoreShippingRates(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can configure shipping rates)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Shipping", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.ShippingCostResponse](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Shipping") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Create new shipping rate record in database
	err := models.AddNewShippingRate(models.DB, *req)
	if err != nil {
		// Shipping rate creation failed (duplicate location or database error)
		log.Printf("Error adding new shipping rate: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to add new shipping rate",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Invalidate location caches to ensure fresh data
	utils.DeleteCacheByPrefix("locations_")
	utils.DeleteCacheByPrefix("locations_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Shipping",
			Description: "Location added successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Location added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200       {object}  map[string]any    "Feedback submitted successfully"
// @Failure      400       {object}  dtos.ErrorResponse      "Invalid feedback data"
// @Router       /api/deliveries/feedback [post]
func SubmitFeedbackHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Decode and parse JSON request body with feedback details
	req, ok := DecodeRequestBody[dtos.DeliveryFeedback](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Save delivery feedback to database for quality tracking
	err := models.AddNewDeliveryFeedback(models.DB, *req)
	if err != nil {
		// Feedback submission failed (invalid delivery ID or database error)
		log.Printf("Error adding new feed back: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to submit delivery feedback",
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Feedback submitted successfully for service improvement
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Feedback submitted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Feedback submitted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func GetDeliveryFeedbacks(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract delivery ID from URL path parameters
	deliveryID := c.Param("delivery_id")

	// Fetch feedback data from database for the specified delivery
	feedback, err := models.GetDeliveryFeedBack(models.DB, deliveryID)
	if err != nil {
		// Handle different error types with appropriate responses
		if err == sql.ErrNoRows {
			// No feedback exists for this delivery
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Users",
					Description: "No delivery feedback found",
					Code:        http.StatusNotFound,
				},
				Message:   "No delivery feedback found",
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
					Description: "Error fetching feedback",
					Code:        http.StatusInternalServerError,
				},
				Message:   "Error fetching feedback",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
		}
		return
	}

	// Return feedback data for delivery quality assessment

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Feedback fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   feedback,
		Message:   "Feedback",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
