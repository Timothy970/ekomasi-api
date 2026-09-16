// Package handlers provides HTTP request handlers for shipping and delivery management.
// This file contains handlers for shipping cost calculation, location management, delivery feedback,
// and related logistics operations. Supports shipping rate configuration, location-based pricing,
// and customer feedback collection for delivery quality assessment.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
func UpdateLocation(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract location ID from URL path parameters
	locationID := c.Param("location_id")
	// Verify user has admin privileges (only admins can update locations)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Shipping", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.UpdateLocation](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Shipping") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}

	// Update location details in database
	if err := models.UpdateLocation(models.DB, *req, locationID); err != nil {
		// Update failed (location not found or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to update location with ID " + locationID,
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
			Description: "Location with ID " + locationID + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Location updated sucessfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func DeleteLocation(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract location ID from URL path parameters
	locationID := c.Param("location_id")
	// Verify user has admin privileges (only admins can delete locations)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Shipping", "")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}

	// Permanently delete location from database
	if err := models.DeleteLocation(models.DB, locationID); err != nil {
		// Deletion failed (location not found, in use, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Shipping",
				Description: "Failed to delete location with ID " + locationID,
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
			Description: "Location with ID " + locationID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Location deleted sucessfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
func DeleteFeedbackHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract feedback ID from URL path parameters
	feedbackID := c.Param("feedback_id")

	// Permanently delete feedback from database
	err := models.DeleteDeliveryFeedback(models.DB, feedbackID)
	if err != nil {
		// Deletion failed (feedback not found or database error)
		log.Printf("Error adding new feed back: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete delivery feedback with ID " + feedbackID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Feedback deleted successfully
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Feedback with ID " + feedbackID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Feedback deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
