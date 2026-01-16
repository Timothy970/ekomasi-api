// Package handlers provides HTTP request handlers for wishlist and gift management.
// This file contains handlers for managing user wishlists, sharing wishlists via email,
// and handling gift registries. Supports adding/removing products, creating multiple wishlists,
// and generating shareable links. Essential for customer engagement and gift-giving features.
package handlers

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/utils"

	"github.com/gorilla/mux"
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
// @Success      200            {object}  dtos.SuccessResponse      "Product added to wishlist"
// @Failure      400            {object}  dtos.ErrorResponse        "Invalid request"
// @Failure      401            {object}  dtos.ErrorResponse        "User not authenticated"
// @Security     BearerAuth
// @Router       /api/wishlist/product [post]
func AddToWishList(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Decode and parse JSON request body with product ID
	item, ok := DecodeRequestBody[dtos.CreateWishlistItem](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (product ID)
	if !utils.ValidateStructAndRespond(item, w, r, requestSummary, start, "Products") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Get user's wishlist or create default "My Wishlist" if none exists (frictionless onboarding)
	wishlistID, err := models.GetOrCreateWishlist(user.ID, "My Wishlist")
	if err != nil {
		// Failed to get or create wishlist
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get or create wishlist for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Add product to wishlist (prevents duplicates at database level)
	err = models.CreateWishListItem(wishlistID, item.ProductID, user.ID)
	if err != nil {
		// Failed to add item (duplicate, product not found, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create wishlist item for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Fetch updated wishlist with all products to return in response
	myWishlist, err := models.GetMyWishlistItems(user.ID)
	if err != nil {
		// Failed to fetch updated wishlist
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlist items for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return success response with updated wishlist containing all products
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product added successfully to the wishlist for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   myWishlist,
		Message:   "Product added successfully to the wishlist",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200            {object}  dtos.SuccessResponse   "Product removed from wishlist"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Failure      404            {object}  dtos.ErrorResponse     "Product not in wishlist"
// @Security     BearerAuth
// @Router       /api/wishlist/product/{product_id} [delete]
func RemoveFromWishList(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusUnauthorized,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Extract product ID from URL path parameters
	productID := mux.Vars(r)["product_id"]

	// Get user's wishlist ID
	wishlistID, err := models.GetWishlistByUserID(user.ID)
	if err != nil {
		// Wishlist not found for user
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlist for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Remove product from wishlist
	err = models.RemoveWishlistItem(wishlistID, productID, user.ID)
	if err != nil {
		// Failed to remove item (product not in wishlist or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to remove wishlist item for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Fetch updated wishlist with remaining products to return in response
	myWishlist, err := models.GetMyWishlistItems(user.ID)
	if err != nil {
		// Failed to fetch updated wishlist
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlist items for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return success response with updated wishlist
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product removed successfully from the wishlist for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   myWishlist,
		Message:   "Product removed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200            {object}  dtos.SuccessResponse   "Wishlists with pagination"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Failure      404            {object}  dtos.ErrorResponse     "No wishlists found"
// @Security     BearerAuth
// @Router       /api/wishlist [get]
func GetAllUserWishList(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Initialize pagination defaults
	limit := 0
	page := 1
	// Get optional wishlist ID filter from URL path
	wishlistID := mux.Vars(r)["wishlist_id"]
	// Parse pagination query parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("size")
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
	wishlists, pagination, err := models.GetAllUserWishList(user.ID, wishlistID, limit, page)
	if err != nil {
		log.Printf("no wishlist:: %s", err)
		// No wishlists found for user
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to get wishlists for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   "WishList was not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Construct response with wishlists and optional pagination metadata
	response := map[string]interface{}{
		"wishlists": wishlists,
	}
	if pagination != nil {
		// Include pagination metadata if available
		response["pagination"] = pagination
	}
	// Return success response with wishlists and pagination
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlists fetched successfully for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "All wishlists",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// CreateWishList creates a new named wishlist for the user.
// Allows users to organize products into multiple themed wishlists (e.g., "Birthday", "Wedding").
//
// @Summary      Create a new wishlist
// @Description  Create a new named wishlist for organizing products
// @Tags         Wishlist
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                 true   "Bearer token"
// @Param        wishlist       body      dtos.CreateWishlist    true   "Wishlist name and details"
// @Success      201            {object}  dtos.SuccessResponse   "Wishlist created successfully"
// @Failure      400            {object}  dtos.ErrorResponse     "Invalid request"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Security     BearerAuth
// @Router       /api/wishlist [post]
func CreateWishList(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Decode and parse JSON request body with wishlist details
	body, ok := DecodeRequestBody[dtos.CreateWishlist](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Create new wishlist in database
	newList, err := models.CreateWishList(*body, user.ID)
	if err != nil {
		// Wishlist creation failed (duplicate name or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create wishlist for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to create wishlist",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return success response with newly created wishlist
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlist created successfully for user with ID " + user.ID,
			Code:        http.StatusCreated,
		},
		Payload:   newList,
		Message:   "Wishlist created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// SendWishlistToShare generates a shareable wishlist link and sends it via email.
// Creates a base64-encoded public link that recipients can view without authentication.
// Sends email with product images, prices, and direct product links.
//
// @Summary      Share wishlist via email
// @Description  Generate shareable wishlist link and send to recipient via email with product details
// @Tags         Wishlist
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                        true   "Bearer token"
// @Param        share          body      dtos.ShareWishlistPayload     true   "Recipient email and message"
// @Success      200            {object}  dtos.SuccessResponse          "Wishlist shared successfully"
// @Failure      400            {object}  dtos.ErrorResponse            "Invalid request"
// @Failure      403            {object}  dtos.ErrorResponse            "Wishlist is private"
// @Failure      404            {object}  dtos.ErrorResponse            "Wishlist not found or empty"
// @Security     BearerAuth
// @Router       /api/wishlist/share [post]
func SendWishlistToShare(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Decode and parse JSON request body with recipient email and message
	req, ok := DecodeRequestBody[dtos.ShareWishlistPayload](r, w, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Validate all required fields (recipient email, sender name, message)
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Auth") {
		// Validation failed, ValidateStructAndRespond already sent error response
		return
	}
	// Fetch user's wishlist with all products
	wishlists, err := models.GetMyWishlistItems(user.ID)

	if err != nil || len(wishlists.Products) == 0 {
		// Wishlist not found or empty (no products to share)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist not found for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   "Wishlist not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Verify wishlist is marked as public (privacy check)
	if !wishlists.IsPublic {
		// Wishlist is private, cannot be shared
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist is private for user with ID " + user.ID,
				Code:        http.StatusForbidden,
			},
			Message:   "This wishlist is private",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Get frontend base URL from environment variables
	baseURL := os.Getenv("FRONT_END_BASE_URL")
	// Generate base64-encoded shareable link (format: wishlistID:userID)
	encodedID := base64.URLEncoding.EncodeToString([]byte(wishlists.WishlistID + ":" + user.ID))
	shareLink := fmt.Sprintf("%sshared-wishlist/%s", baseURL, encodedID)
	// Prepare email subject
	subject := "Check out my wishlist!"
	wishlistItems := []utils.WishlistItem{}
	// Build wishlist items array for email template with product details
	for _, product := range wishlists.Products {
		item := utils.WishlistItem{
			Title:      product.Name,
			ImageURL:   product.Images[0].URL,
			Price:      fmt.Sprintf("$%.2f", product.Price),
			ProductURL: fmt.Sprintf("%sproducts/%s", baseURL, product.ID),
		}
		wishlistItems = append(wishlistItems, item)
	}

	// Build sender details string with email and phone if available
	senderDetails := ""

	if user.Email != "" {
		senderDetails += "Email: " + user.Email + " "
	}

	if user.Phone != "" {
		senderDetails += " Phone: " + user.Phone
	}

	// Generate HTML email body with product images, prices, and links
	htmlBody, err := utils.GenerateWishlistEmailHTML(req.SenderName, senderDetails, *req.Message, shareLink, wishlistItems)
	if err != nil {
		log.Printf("failed to generate email HTML: %v", err)
		return
	}
	// Send email to recipient with wishlist details
	notification.SendEmail(req.Email, subject, htmlBody)

	// Return success response confirming wishlist was shared
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Share link generated successfully" + shareLink,
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Wishlist shared successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ReceiceWishlistShared retrieves a public wishlist using a shareable link.
// Decodes base64-encoded wishlist ID and returns wishlist with products.
// Public endpoint - no authentication required for viewing shared wishlists.
//
// @Summary      Get shared wishlist
// @Description  Retrieve public wishlist and its products using shareable link ID
// @Tags         Wishlist
// @Produce      json
// @Param        wishlist_id  path      string                 true  "Base64-encoded wishlist ID"
// @Success      200          {object}  dtos.SuccessResponse   "Shared wishlist details"
// @Failure      400          {object}  dtos.ErrorResponse     "Invalid wishlist link"
// @Failure      403          {object}  dtos.ErrorResponse     "Wishlist is private"
// @Failure      404          {object}  dtos.ErrorResponse     "Wishlist not found"
// @Router       /api/wishlist/share/{wishlist_id} [get]
func ReceiceWishlistShared(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract encoded wishlist ID from URL path parameters
	vars := mux.Vars(r)
	encodedID := vars["wishlist_id"]

	// Decode base64-encoded wishlist link
	decodedBytes, err := base64.URLEncoding.DecodeString(encodedID)
	if err != nil {
		// Base64 decoding failed, invalid link format
		// Error response will be sent below
	}
	// Convert decoded bytes to string and split by colon separator
	decoded := string(decodedBytes)
	parts := strings.SplitN(decoded, ":", 2)
	if len(parts) != 2 {
		// Invalid format, expected "wishlistID:userID"
		// Error response will be sent below
	}
	// Extract wishlist ID from decoded link (userID not needed for retrieval)
	wishlistID := parts[0]
	// userID := parts[1]
	if err != nil {
		// Send error response for invalid link
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Invalid wishlist link",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid wishlist link",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Fetch wishlist details from database
	wishlists, err := models.GetWishlistByID(wishlistID)

	if err != nil || len(wishlists) == 0 {
		// Wishlist not found or empty
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist not found or is empty with ID " + wishlistID,
				Code:        http.StatusNotFound,
			},
			Message:   "Wishlist not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Get first wishlist from results
	wishlist := wishlists[0]
	// Verify wishlist is marked as public (privacy check)
	if !wishlist.IsPublic {
		// Wishlist is private, cannot be viewed via public link
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist with ID " + wishlistID + " is private",
				Code:        http.StatusForbidden,
			},
			Message:   "This wishlist is private",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return wishlist with all products to public viewer
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Shared wishlist fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   wishlist,
		Message:   "Shared wishlist",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteWishList removes a wishlist and all its items.
// Only the wishlist owner can delete their wishlists.
//
// @Summary      Delete wishlist
// @Description  Delete a wishlist and all its items (owner only)
// @Tags         Wishlist
// @Produce      json
// @Param        Authorization  header    string                 true  "Bearer token"
// @Param        wishlist_id    path      string                 true  "Wishlist ID to delete"
// @Success      200            {object}  dtos.SuccessResponse   "Wishlist deleted successfully"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Failure      404            {object}  dtos.ErrorResponse     "Wishlist not found"
// @Security     BearerAuth
// @Router       /api/wishlist/{wishlist_id} [delete]
func DeleteWishList(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Extract wishlist ID from URL path parameters
	wishlistID := mux.Vars(r)["wishlist_id"]
	// Delete wishlist and all its items (verifies ownership)
	err := models.DeleteWishList(wishlistID, user.ID)
	if err != nil {
		// Deletion failed (wishlist not found, not owned by user, or database error)
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to delete wishlist with ID " + wishlistID + " for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Return success response confirming deletion
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlist with ID " + wishlistID + " deleted successfully for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Wishlist deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success      200            {object}  dtos.SuccessResponse   "User's wishlist with products"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Failure      500            {object}  dtos.ErrorResponse     "Failed to retrieve wishlist"
// @Security     BearerAuth
// @Router       /api/wishlist/me [get]
func GetMyWishList(w http.ResponseWriter, r *http.Request) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(r)
	// Extract authenticated user from request context
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		// User not authenticated or token invalid
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: noUser,
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Fetch user's primary wishlist with all product details
	myWishlist, err := models.GetMyWishlistItems(user.ID)
	if err != nil {
		// Failed to retrieve wishlist
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to retrieve wishlist items for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	// Return wishlist with all product details
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlist items retrieved successfully for user with ID " + user.ID,
			Code:        http.StatusOK,
		},
		Payload:   myWishlist,
		Message:   "My wishlists",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
