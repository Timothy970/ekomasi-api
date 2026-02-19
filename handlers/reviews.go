// Package handlers provides HTTP request handlers for product review management.
// This file contains handlers for customer reviews and ratings, including creating reviews,
// viewing product reviews with filtering/sorting, moderating reviews (admin), and managing
// review visibility. Reviews help build trust and provide valuable product feedback.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"time"

	"github.com/gorilla/mux"
)

var productWithID = "Product with ID "

// GetReview retrieves a specific review for a product by review ID.
// Provides detailed review information including rating, comment, reviewer details, and timestamps.
// Supports pagination if multiple reviews need to be returned.
//
// @Summary      Get specific product review
// @Description  Retrieve detailed information about a specific review for a product
// @Tags         Reviews
// @Produce      json
// @Param        product_id  path      string  true   "Product ID"
// @Param        review_id   path      string  true   "Review ID"
// @Param        page        query     int     false  "Page number (default: 1)"
// @Param        size        query     int     false  "Page size (default: 10)"
// @Success      200         {object}  map[string]interface{}  "Review details with pagination"
// @Failure      500         {object}  dtos.ErrorResponse      "Internal server error"
// @Router       /api/products/{product_id}/reviews/{review_id} [get]
func GetReview(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract product ID and review ID from URL path parameters
	productID := mux.Vars(r)["product_id"]
	reviewID := mux.Vars(r)["review_id"]
	// Parse pagination parameters from query string
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	// Fetch specific review(s) from database for the product
	reviews, pagination, err := models.GetProductReview(models.DB, productID, reviewID, limit, page)
	if err != nil {
		// Database query failed, log and return error response
		log.Printf("error getting reviews:::%v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetching product reviews for product with product id " + productID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})

		return
	}

	// Build response payload with reviews data
	// TODO1: Consider caching review data in Redis for better performance
	response := map[string]interface{}{
		"reviews": reviews,
	}
	// Include pagination metadata if available
	if pagination != nil {
		response["pagination"] = pagination
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product reviews for product ID " + productID + " retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Product reviews",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// GetProductReviews retrieves all reviews for a specific product with filtering and sorting.
// Supports filtering by rating (1-5 stars), sorting (newest, oldest, highest, lowest rated),
// and pagination for large review sets. Essential for product pages and customer research.
//
// @Summary      Get all product reviews
// @Description  Retrieve paginated list of all reviews for a product with optional filtering and sorting
// @Tags         Reviews
// @Produce      json
// @Param        product_id  path      string  true   "Product ID"
// @Param        page        query     int     false  "Page number (default: 1)"
// @Param        size        query     int     false  "Page size (default: 10)"
// @Param        sort_by     query     string  false  "Sort order (newest, oldest, highest_rated, lowest_rated)"
// @Param        ratings     query     int     false  "Filter by rating (1-5 stars)"
// @Success      200         {object}  map[string]interface{}  "Reviews list with pagination"
// @Failure      404         {object}  dtos.ErrorResponse      "Product not found"
// @Failure      500         {object}  dtos.ErrorResponse      "Internal server error"
// @Router       /api/products/{product_id}/reviews [get]
func GetProductReviews(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract product ID from URL path parameters
	productID := mux.Vars(r)["product_id"]
	// Parse pagination parameters from query string
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	// Extract optional sorting parameter (newest, oldest, highest_rated, lowest_rated)
	sortBy := r.URL.Query().Get("sort_by")
	// Extract optional rating filter parameter (1-5 stars)
	ratingStr := r.URL.Query().Get("ratings")
	rating, _ := strconv.Atoi(ratingStr)
	// Fetch all reviews for product from database with filters and sorting
	reviews, pagination, err := models.GetProductReviews(models.DB, productID, sortBy, rating, limit, page)
	if err != nil {
		// Handle different error types with appropriate responses
		if err == sql.ErrNoRows {
			// Product not found in database
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: productWithID + productID + " is not found",
					Code:        http.StatusNotFound,
				},
				Message:   "Product not found",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			// Database query failed, log and return error response
			log.Printf("error getting reviews:::%v", err)
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Error fetching product reviews for product with ID " + productID,
					Code:        http.StatusInternalServerError,
				},
				Message:   "Error fetching product reviews",
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		}
		return
	}

	// Build response payload with reviews data
	// TODO1: Consider caching review data in Redis for better performance
	response := map[string]interface{}{
		"reviews": reviews,
	}
	// Include pagination metadata if available
	if pagination != nil {
		response["pagination"] = pagination
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product reviews for product ID " + productID + " retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "All reviews for product",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CreateReview allows authenticated users to submit a review for a product.
// Reviews include rating (1-5 stars), comment, and optional media attachments.
// Users typically can only review products they have purchased (verification may be enforced).
//
// @Summary      Create product review
// @Description  Submit a review with rating and comment for a product
// @Tags         Reviews
// @Accept       json
// @Produce      json
// @Param        product_id  path      string               true  "Product ID"
// @Param        review      body      dtos.ReviewRequest   true  "Review details"
// @Success      201         {object}  dtos.ReviewResponse  "Review created successfully"
// @Failure      400         {object}  dtos.ErrorResponse   "Invalid request or validation failed"
// @Failure      401         {object}  dtos.ErrorResponse   "User not authenticated"
// @Failure      500         {object}  dtos.ErrorResponse   "Internal server error"
// @Security     BearerAuth
// @Router       /api/products/{product_id}/reviews [post]
func CreateReview(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract product ID from URL path parameters
	productID := mux.Vars(r)["product_id"]
	// Extract authenticated user from request context
	authuser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated, return unauthorized error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "User not validated or authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.ReviewRequest](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Associate review with authenticated user
	req.UserID = authuser.ID
	// Validate all required fields in the request (rating, comment, etc.)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Verify that the product exists before allowing review submission
	err := models.IsProductThere(models.DB, productID)
	if err != nil {
		// Product not found or database error
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error fetching product reviews for product with ID " + productID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})

		return
	}
	// Create new review in database (may include duplicate check)
	review, err := models.AddNewReview(models.DB, *req, productID)
	if err != nil {
		// Review creation failed (duplicate, invalid data, or database error)
		log.Printf("Error adding new product review: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error adding new review for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate Redis cache for the product reviews to ensure fresh data
	ctx := context.Background()
	cacheKey := fmt.Sprintf("reviews_%s", productID)
	if err := Redis.Del(ctx, cacheKey).Err(); err != nil {
		// Cache invalidation failed but review was created successfully
		log.Printf("Failed to invalidate review cache: %v", err)
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Review added successfully for product ID " + productID,
			Code:        http.StatusCreated,
		},
		Payload:   review,
		Message:   "Review added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

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
func UpdateReview(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Decode and parse JSON request body
	req, ok := DecodeRequestBody[dtos.UpdateReview](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields in the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract product ID and review ID from URL path parameters
	productID := mux.Vars(r)["product_id"]
	reviewID := mux.Vars(r)["review_id"]

	// Update review in database (admin moderation/content update)
	err := models.UpdateReview(models.DB, *req, reviewID, productID)
	if err != nil {
		// Update failed (review not found or database error)
		log.Printf("Error moderating review: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error moderating review for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate all review caches in Redis to ensure fresh data
	// Scan for all keys matching the review cache pattern
	pattern := "reviews_"

	iter := Redis.Scan(r.Context(), 0, pattern, 0).Iterator()
	for iter.Next(r.Context()) {
		// Delete each matching cache key
		if err := Redis.Del(r.Context(), iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete review cache key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("Failed to scan review cache keys: %v", err)
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Review moderated successfully for product ID " + productID,
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Review updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func DeleteReview(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	ctx := r.Context()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Verify user has admin privileges (only admins can delete reviews)
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Products", "products.delete")
	if !ok {
		// Authorization failed, RequireAdmin already sent error response
		return
	}
	// Extract product ID and review ID from URL path parameters
	productID := mux.Vars(r)["product_id"]
	reviewID := mux.Vars(r)["review_id"]
	// Verify product exists before attempting to delete review
	product, err := models.GetProductByID(models.DB, productID)
	if err != nil || product == nil {
		// Handle different error types with appropriate responses
		if err == sql.ErrNoRows {
			// Product not found in database
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: productWithID + productID + " not found",
					Code:        http.StatusNotFound,
				},
				Message:   productNotFound,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
			// Database query failed
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: "Error fetching product reviews for product ID " + productID,
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
	// Permanently delete review from database
	err = models.DeleteReview(models.DB, reviewID)
	if err != nil {
		// Deletion failed (review not found or database error)
		log.Printf("Error deleteing review: %v", err)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Error deleting review for product ID " + productID,
				Code:        http.StatusBadRequest,
			},
			Message:   fmt.Sprintf("%s", err),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
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

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Review deleted successfully for product ID " + productID,
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Review deleted succesfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
