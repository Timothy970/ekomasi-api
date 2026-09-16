// Package handlers provides HTTP request handlers for wishlist and gift management.
// This file contains handlers for managing user wishlists, sharing wishlists via email,
// and handling gift registries. Supports adding/removing products, creating multiple wishlists,
// and generating shareable links. Essential for customer engagement and gift-giving features.
package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

// noUser is the standard error message for missing user authentication
var noUser = "User is not validated"

// This file provides handlers for Wishlist and Gifts endpoints.
// All endpoints require user authentication for creation and modifications.

// AddToWishList adds a product to the user's wishlist.
// Auto-creates a default wishlist if user doesn't have one (frictionless experience).
// Returns updated wishlist with all products after successful addition.
//
// @Summary      Add product to wishlist
// @Description  Add a product to the user's wishlist (creates default wishlist if needed)
// @Tags         Wishlist
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                    true   "Bearer token"
// @Param        wishlist       body      dtos.CreateWishlistItem   true   "Wishlist item to add"
// @Success      200            {object}  map[string]any      "Product added to wishlist"
// @Failure      400            {object}  dtos.ErrorResponse        "Invalid request"
// @Failure      401            {object}  dtos.ErrorResponse        "User not authenticated"
// @Security     BearerAuth
// @Router       /api/wishlist/product [post]
func AddToWishList(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Decode and parse JSON request body with product ID
	item, ok := DecodeRequestBody[dtos.CreateWishlistItem](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (product ID)
	if !utils.ValidateGinStructAndRespond(item, c, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Get user's wishlist or create default "My Wishlist" if none exists (frictionless onboarding)
	wishlistID, err := models.GetOrCreateWishlist(models.DB, user.ID, "My Wishlist")
	if err != nil {
		// Failed to get or create wishlist
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get or create wishlist for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Add product to wishlist (prevents duplicates at database level)
	err = models.CreateWishListItem(models.DB, wishlistID, item.ProductID, user.ID)
	if err != nil {
		// Failed to add item (duplicate, product not found, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create wishlist item for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Fetch updated wishlist with all products to return in response
	myWishlist, err := models.GetMyWishlistItems(models.DB, user.ID)
	if err != nil {
		// Failed to fetch updated wishlist
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlist items for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Return success response with updated wishlist containing all products
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product added successfully to the wishlist for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   myWishlist,
		Message:   "Product added successfully to the wishlist",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// RemoveFromWishList removes a product from the user's wishlist.
// Returns updated wishlist with remaining products after successful removal.
//
// @Summary      Remove product from wishlist
// @Description  Remove a product from the user's wishlist by product ID
// @Tags         Wishlist
// @Produce      json
// @Param        Authorization  header    string                 true   "Bearer token"
// @Param        product_id     path      string                 true   "Product ID to remove"
// @Success      200            {object}  map[string]any   "Product removed from wishlist"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Failure      404            {object}  dtos.ErrorResponse     "Product not in wishlist"
// @Security     BearerAuth
// @Router       /api/wishlist/product/{product_id} [delete]
func RemoveFromWishList(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusUnauthorized,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Extract product ID from URL path parameters
	productID := c.Param("product_id")

	// Get user's wishlist ID
	wishlistID, err := models.GetWishlistByUserID(models.DB, user.ID)
	if err != nil {
		// Wishlist not found for user
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlist for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Remove product from wishlist
	err = models.RemoveWishlistItem(models.DB, wishlistID, productID, user.ID)
	if err != nil {
		// Failed to remove item (product not in wishlist or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to remove wishlist item for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Fetch updated wishlist with remaining products to return in response
	myWishlist, err := models.GetMyWishlistItems(models.DB, user.ID)
	if err != nil {
		// Failed to fetch updated wishlist
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlist items for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Return success response with updated wishlist
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product removed successfully from the wishlist for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   myWishlist,
		Message:   "Product removed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetAllUserWishList retrieves all wishlists for the authenticated user.
// Supports pagination for users with many wishlist items.
// Returns wishlists with product details and pagination metadata.
//
// @Summary      Get all wishlists for user
// @Description  Returns all wishlists with products for the authenticated user with pagination
// @Tags         Wishlist
// @Produce      json
// @Param        Authorization  header    string                 true   "Bearer token"
// @Param        page           query     int                    false  "Page number (default: 1)"
// @Param        size           query     int                    false  "Page size (default: 10)"
// @Success      200            {object}  map[string]any   "Wishlists with pagination"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Failure      404            {object}  dtos.ErrorResponse     "No wishlists found"
// @Security     BearerAuth
// @Router       /api/wishlist [get]
func GetAllUserWishList(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Initialize pagination defaults
	limit := 0
	page := 1
	// Get optional wishlist ID filter from URL path
	wishlistID := c.Param("wishlist_id")
	// Parse pagination query parameters
	pageStr := c.Query("page")
	limitStr := c.Query("size")
	if limitStr != "" {
		// Convert size parameter to integer
		limit, _ = strconv.Atoi(limitStr)
	} else {
		// Default page size is 10 items
		limit = 10
	}
	if pageStr != "" {
		// Convert page parameter to integer
		page, _ = strconv.Atoi(pageStr)
	}
	// Fetch user's wishlists with pagination
	wishlists, pagination, err := models.GetAllUserWishList(models.DB, user.ID, wishlistID, limit, page)
	if err != nil {
		log.Printf("no wishlist:: %s", err)
		// No wishlists found for user
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlists for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   "WishList was not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Construct response with wishlists and optional pagination metadata
	response := map[string]any{
		"wishlists": wishlists,
	}
	if pagination != nil {
		// Include pagination metadata if available
		response["pagination"] = pagination
	}
	// Return success response with wishlists and pagination
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlists fetched successfully for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "All wishlists",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
