// Package handlers provides HTTP request handlers for wishlist and gift management.
// This file contains handlers for managing user wishlists, sharing wishlists via email,
// and handling gift registries. Supports adding/removing products, creating multiple wishlists,
// and generating shareable links. Essential for customer engagement and gift-giving features.
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

// DeleteWishList removes a wishlist and all its items.
// Only the wishlist owner can delete their wishlists.
//
// @Summary      Delete wishlist
// @Description  Delete a wishlist and all its items (owner only)
// @Tags         Wishlist
// @Produce      json
// @Param        Authorization  header    string                 true  "Bearer token"
// @Param        wishlist_id    path      string                 true  "Wishlist ID to delete"
// @Success      200            {object}  map[string]any   "Wishlist deleted successfully"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Failure      404            {object}  dtos.ErrorResponse     "Wishlist not found"
// @Security     BearerAuth
// @Router       /api/wishlist/{wishlist_id} [delete]
func DeleteWishList(c *gin.Context) {
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
	// Extract wishlist ID from URL path parameters
	wishlistID := c.Param("wishlist_id")
	// Delete wishlist and all its items (verifies ownership)
	err := models.DeleteWishList(models.DB, wishlistID, user.ID)
	if err != nil {
		// Deletion failed (wishlist not found, not owned by user, or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete wishlist with ID " + wishlistID + " for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response confirming deletion
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlist with ID " + wishlistID + " deleted successfully for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Wishlist deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetMyWishList retrieves the user's primary wishlist with all products.
// Returns the default "My Wishlist" with complete product details.
// Essential for displaying user's saved items.
//
// @Summary      Get my wishlist
// @Description  Retrieve user's primary wishlist with all product details
// @Tags         Wishlist
// @Produce      json
// @Param        Authorization  header    string                 true  "Bearer token"
// @Success      200            {object}  map[string]any   "User's wishlist with products"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Failure      500            {object}  dtos.ErrorResponse     "Failed to retrieve wishlist"
// @Security     BearerAuth
// @Router       /api/wishlist/me [get]
func GetMyWishList(c *gin.Context) {
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
	// Fetch user's primary wishlist with all product details
	myWishlist, err := models.GetMyWishlistItems(models.DB, user.ID)
	if err != nil {
		// Failed to retrieve wishlist
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to retrieve wishlist items for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Return wishlist with all product details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlist items retrieved successfully for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   myWishlist,
		Message:   "My wishlists",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
