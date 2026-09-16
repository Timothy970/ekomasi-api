// Package handlers provides HTTP request handlers for product review management.
// This file contains handlers for customer reviews and ratings, including creating reviews,
// viewing product reviews with filtering/sorting, moderating reviews (admin), and managing
// review visibility. Reviews help build trust and provide valuable product feedback.
package handlers

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"time"
)

// UpdateReview allows admins to moderate and update a review.
// Admins can edit review content, change visibility status, or flag inappropriate reviews.
// This is used for content moderation and maintaining review quality standards.
//
// @Summary      Update/moderate product review
// @Description  Update review details or moderate review content (admin only)
// @Tags         Reviews
// @Accept       json
// @Produce      json
// @Param        product_id  path      string              true  "Product ID"
// @Param        review_id   path      string              true  "Review ID"
// @Param        update      body      dtos.UpdateReview   true  "Review update details"
// @Success      200         {object}  map[string]interface{}  "Review updated successfully"
// @Failure      400         {object}  dtos.ErrorResponse    "Invalid request or update failed"
// @Failure      401         {object}  dtos.ErrorResponse    "User not authorized (admin required)"
// @Security     BearerAuth
// @Router       /api/admin/products/{product_id}/reviews/{review_id} [patch]
func UpdateReview(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.UpdateReview](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract product ID and review ID from URL path parameters
	productID := c.Param("product_id")
	reviewID := c.Param("review_id")

	// Update review in database (admin moderation/content update)
	err := models.UpdateReview(models.DB, *req, reviewID, productID)
	if err != nil {
		// Update failed (review not found or database error)
		log.Printf("Error moderating review: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error moderating review for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate all review caches in Redis to ensure fresh data
	// Scan for all keys matching the review cache pattern
	pattern := "reviews_"

	iter := Redis.Scan(c.Request.Context(), 0, pattern, 0).Iterator()
	for iter.Next(c.Request.Context()) {
		// Delete each matching cache key
		if err := Redis.Del(c.Request.Context(), iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete review cache key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("Failed to scan review cache keys: %v", err)
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Review moderated successfully for product ID " + productID,
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Review updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteReview permanently removes a review from the system.
// Admin-only operation typically used for removing inappropriate, spam, or fake reviews.
// Deletion is permanent and cannot be undone - use with caution.
//
// @Summary      Delete product review
// @Description  Permanently delete a review from the system (admin only)
// @Tags         Reviews
// @Produce      json
// @Param        product_id  path      string                true  "Product ID"
// @Param        review_id   path      string                true  "Review ID"
// @Success      200         {object}  map[string]interface{}  "Review deleted successfully"
// @Failure      400         {object}  dtos.ErrorResponse    "Review not found or deletion failed"
// @Failure      401         {object}  dtos.ErrorResponse    "User not authorized (admin required)"
// @Failure      404         {object}  dtos.ErrorResponse    "Product not found"
// @Security     BearerAuth
// @Router       /api/admin/products/{product_id}/reviews/{review_id} [delete]
func DeleteReview(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	ctx := c.Request.Context()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Verify user has admin privileges (only admins can delete reviews)
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Products", "products.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract product ID and review ID from URL path parameters
	productID := c.Param("product_id")
	reviewID := c.Param("review_id")
	// Verify product exists before attempting to delete review
	product, err := models.GetProductByID(models.DB, productID)
	if err != nil || product == nil {
		// Handle different error types with appropriate responses
		if err == sql.ErrNoRows {
			// Product not found in database
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: productWithID + productID + " not found",
					Code:        http.StatusNotFound,
				},
				Message:   productNotFound,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary})
		} else {
			// Database query failed
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Error fetching product reviews for product ID " + productID,
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
	// Permanently delete review from database
	err = models.DeleteReview(models.DB, reviewID)
	if err != nil {
		// Deletion failed (review not found or database error)
		log.Printf("Error deleteing review: %v", err)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error deleting review for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate all review caches in Redis to ensure fresh data
	// Scan for all keys matching the review cache pattern
	pattern := "reviews_"

	iter := Redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		// Delete each matching cache key
		if err := Redis.Del(ctx, iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete review cache key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("Failed to scan review cache keys: %v", err)
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Review deleted successfully for product ID " + productID,
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Review deleted succesfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
