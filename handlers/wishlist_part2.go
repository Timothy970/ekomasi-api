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
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/notification"
	"ekomasi_backend/utils"
)

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
// @Success      201            {object}  map[string]interface{}   "Wishlist created successfully"
// @Failure      400            {object}  dtos.ErrorResponse     "Invalid request"
// @Failure      401            {object}  dtos.ErrorResponse     "User not authenticated"
// @Security     BearerAuth
// @Router       /api/wishlist [post]
func CreateWishList(c *gin.Context) {
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
	// Decode and parse JSON request body with wishlist details
	body, ok := DecodeRequestBody[dtos.CreateWishlist](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Create new wishlist in database
	newList, err := models.CreateWishList(models.DB, *body, user.ID)
	if err != nil {
		// Wishlist creation failed (duplicate name or database error)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to create wishlist for user with ID " + user.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to create wishlist",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response with newly created wishlist
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Wishlist created successfully for user with ID " + user.ID,
			Code:        http.StatusCreated,
		},
		Payload:   newList,
		Message:   "Wishlist created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200            {object}  map[string]interface{}          "Wishlist shared successfully"
// @Failure      400            {object}  dtos.ErrorResponse            "Invalid request"
// @Failure      403            {object}  dtos.ErrorResponse            "Wishlist is private"
// @Failure      404            {object}  dtos.ErrorResponse            "Wishlist not found or empty"
// @Security     BearerAuth
// @Router       /api/wishlist/share [post]
func SendWishlistToShare(c *gin.Context) {
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
	// Decode and parse JSON request body with recipient email and message
	req, ok := DecodeRequestBody[dtos.ShareWishlistPayload](c, requestSummary, start)
	if !ok {
		// Request body parsing failed, DecodeRequestBody already sent error response
		return
	}
	// Fetch user's wishlist with all products
	wishlists, err := models.GetMyWishlistItems(models.DB, user.ID)

	if err != nil || len(wishlists.Products) == 0 {
		// Wishlist not found or empty (no products to share)
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist not found for user with ID " + user.ID,
				Code:        http.StatusNotFound,
			},
			Message:   "Wishlist not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Verify wishlist is marked as public (privacy check)
	if !wishlists.IsPublic {
		// Wishlist is private, cannot be shared
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist is private for user with ID " + user.ID,
				Code:        http.StatusForbidden,
			},
			Message:   "This wishlist is private",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
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
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Share link generated successfully" + shareLink,
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Wishlist shared successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success      200          {object}  map[string]interface{}   "Shared wishlist details"
// @Failure      400          {object}  dtos.ErrorResponse     "Invalid wishlist link"
// @Failure      403          {object}  dtos.ErrorResponse     "Wishlist is private"
// @Failure      404          {object}  dtos.ErrorResponse     "Wishlist not found"
// @Router       /api/wishlist/share/{wishlist_id} [get]
func ReceiceWishlistShared(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract encoded wishlist ID from URL path parameters
	encodedID := c.Param("wishlist_id")

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
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Invalid wishlist link",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid wishlist link",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch wishlist details from database
	wishlistsByID, err := models.GetWishlistByID(models.DB, wishlistID)

	if err != nil || len(wishlistsByID) == 0 {
		// Wishlist not found or empty
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist not found or is empty with ID " + wishlistID,
				Code:        http.StatusNotFound,
			},
			Message:   "Wishlist not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Get first wishlist from results
	wishlist := wishlistsByID[0]
	// Verify wishlist is marked as public (privacy check)
	if !wishlist.IsPublic {
		// Wishlist is private, cannot be viewed via public link
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Wishlist with ID " + wishlistID + " is private",
				Code:        http.StatusForbidden,
			},
			Message:   "This wishlist is private",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Return wishlist with all products to public viewer
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Shared wishlist fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   wishlist,
		Message:   "Shared wishlist",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
