package handlers

import (
	"adenzo_backend/dtos"
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

// GetReviews godoc
// @Summary      Product Reviews
// @Description  Get all reviews for a specific product
// @Tags         Reviews
// @Produce      json
// @Param        product_id  query     string  true  "Product ID"
// @Success      200         {array}   dtos.ReviewResponse
// @Failure      400         {object}  map[string]string
// @Failure      404         {object}  map[string]string
// @Failure      500         {object}  map[string]string
// @Router       /api/reviews [get]
func GetReview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST

	requestSummary := utils.GetRequestSummary(r)
	// ctx := r.Context()
	productID := mux.Vars(r)["product_id"]
	reviewID := mux.Vars(r)["review_id"]
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	reviews, pagination, err := models.GetProductReview(productID, reviewID, limit, page)
	if err != nil {
		if err == sql.ErrNoRows {
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

	// Cache result
	// if jsonBytes, err := json.Marshal(reviews); err == nil {
	// 	Redis.Set(ctx, cacheKey, jsonBytes, time.Hour)
	// }
	response := map[string]interface{}{
		"reviews": reviews,
	}
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
func GetProductReviews(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST

	requestSummary := utils.GetRequestSummary(r)
	// ctx := r.Context()
	productID := mux.Vars(r)["product_id"]
	page, limit := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	sortBy := r.URL.Query().Get("sort_by")
	ratingStr := r.URL.Query().Get("ratings")
	rating, _ := strconv.Atoi(ratingStr)
	reviews, pagination, err := models.GetProductReviews(productID, sortBy, rating, limit, page)
	if err != nil {
		if err == sql.ErrNoRows {
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

	// Cache result
	// if jsonBytes, err := json.Marshal(reviews); err == nil {
	// 	Redis.Set(ctx, cacheKey, jsonBytes, time.Hour)
	// }
	response := map[string]interface{}{
		"reviews": reviews,
	}
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

// CreateReview godoc
// @Summary      Add Product Review(s)
// @Description  Submit one or more reviews for a product.
// @Tags         Reviews
// @Accept       json
// @Produce      json
// @Param        reviews  body      []dtos.ReviewRequest  true  "List of reviews to add"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/reviews [post]
func CreateReview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	productID := mux.Vars(r)["product_id"]
	req, ok := DecodeRequestBody[dtos.ReviewRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}
	//check if product exists
	product, err := models.GetProductByID(productID)
	if err != nil || product == nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Products",
					Description: productWithID + productID + " has not been found",
					Code:        http.StatusNotFound,
				},
				Message:   productNotFound,
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
		} else {
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
		}
		return
	}
	review, err := models.AddNewReview(*req, productID)
	if err != nil {
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

	// Invalidate Redis cache for the product
	ctx := context.Background()
	cacheKey := fmt.Sprintf("reviews_%s", productID)
	if err := Redis.Del(ctx, cacheKey).Err(); err != nil {
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

// update review
// UpdateReview godoc
// @Summary      Update Product Review
// @Description  Update review for a product.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Success      201     {object}  map[string]interface{}
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/admin/products/{product_id}reviews/{review_id} [PATCH]
func UpdateReview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateReview](r, w, requestSummary, start)
	if !ok {
		return
	}
	// ctx := r.Context()
	productID := mux.Vars(r)["product_id"]
	reviewID := mux.Vars(r)["review_id"]
	//check if product exists
	product, err := models.GetProductByID(productID)
	if err != nil || product == nil {
		if err == sql.ErrNoRows {
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
	err = models.UpdateReview(*req, reviewID)
	if err != nil {
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

	// Invalidate Redis cache for the product
	pattern := "reviews_"

	iter := Redis.Scan(r.Context(), 0, pattern, 0).Iterator()
	for iter.Next(r.Context()) {
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
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Review moderated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Delete a review
// UpdateReview godoc
// @Summary      Delete Product Review
// @Description  Delete review for a product.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Success      201     {object}  map[string]interface{}
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/admin/products/{product_id}reviews/{review_id} [PATCH]
func DeleteReview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Products")
	if !ok {
		return
	}
	productID := mux.Vars(r)["product_id"]
	reviewID := mux.Vars(r)["review_id"]
	//check if product exists
	product, err := models.GetProductByID(productID)
	if err != nil || product == nil {
		if err == sql.ErrNoRows {
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
	err = models.DeleteReview(reviewID)
	if err != nil {
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

	// Invalidate Redis cache for the product
	pattern := "reviews_"

	iter := Redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
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
