package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ekomasi_backend/dtos"
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
)

func GetUserBasedProductsRecommendations(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context and not validated",
				Code:        http.StatusInternalServerError,
			},
			Message:   noUser,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Parse pagination params
	page := 1
	limit := 10
	pageStr := c.Query("page")
	limitStr := c.Query("size")
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}

	// Get categories from purchase history
	purchaseCats, err := models.GetPurchasedCategories(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch purchased categories for user with ID " + authUser.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Get categories from wishlist
	wishlistCats, err := models.GetWishlistCategories(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch wishlist categories for user with ID " + authUser.ID,
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Merge categories
	categories := append(purchaseCats, wishlistCats...)

	// Fetch recommended products based on merged categories
	recommended, pagination, err := models.GetProductsByCategories(models.DB, categories, page, limit)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to fetch recommended products",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with recommended products
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Recommended products fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"products":   recommended,
			"pagination": pagination,
		},
		Message:   "Recommended products fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// AddSubscriber adds a new subscriber to the mailing list.
//
// @Summary      Create a subscriber
// @Description  Add a new email to the subscriber list.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        subscriber  body      dtos.Subscriber  true  "Subscriber Details"
// @Success      201         {object}  map[string]any "Subscriber created successfully"
// @Failure      400         {object}  map[string]string      "Invalid request payload"
// @Failure      404         {object}  map[string]string      "Resource not found"
// @Failure      500         {object}  map[string]string      "Internal server error"
// @Router       /api/subscribe [post]
func AddSubscriber(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Decode request body
	input, ok := DecodeRequestBody[dtos.Subscriber](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the request payload
	if !utils.ValidateGinStructAndRespond(input, c, requestSummary, start, "Users") {
		return
	}

	// Create subscriber in database
	err := models.CreateSubscribers(models.DB, input.Email)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to create subscriber",
				Code:        http.StatusNotFound,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with success message
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "Subscriber added successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Subscriber added successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeactivateMyAccount deactivates the authenticated user's account.
//
// @Summary      Deactivate my account
// @Description  Deactivate the account of the currently authenticated user.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      dtos.RegisterRequest  true  "User Deactivation Details"
// @Success      201   {object}  map[string]any "Account deactivated successfully"
// @Failure      400   {object}  map[string]string      "Invalid request payload or missing mandatory fields"
// @Failure      401   {object}  map[string]string      "Unauthorized"
// @Failure      500   {object}  map[string]string      "Internal server error"
// @Router       /api/admin/users [delete]
func DeactivateMyAccount(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	authUser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not authenticated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	err := models.DeactivateUserByID(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to deactivate user account",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Response
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "User deactivated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User deactivated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// DeleteUser deletes the currently authenticated user's account.
//
// @Summary      Delete User
// @Description  Delete the account of the currently authenticated user.
// @Tags         Users
// @Produce      json
// @Success      200  {object}  map[string]any "User details fetched successfully"
// @Failure      401  {object}  map[string]string      "Unauthorized"
// @Failure      500  {object}  map[string]string      "Internal server error"
// @Router       /api/user/me [delete]
func DeleteUser(c *gin.Context) {
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Get authenticated user from context
	authUser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not authenticated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Fetch user details from database
	err := models.DeleteUserByID(models.DB, authUser.ID)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "Failed to delete user with ID " + authUser.ID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Respond with user details
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Users",
			Description: "User with ID " + authUser.ID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "User deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// VerifyUserUpdateHandler finalizes the user profile update after OTP verification.
//
// @Summary      Verify user update
// @Description  Verify the OTP sent to the new email/phone to finalize the profile update.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        otp  body      map[string]string  true  "OTP verification"
// @Success      200  {object}  map[string]any "Profile updated successfully"
// @Failure      400  {object}  map[string]string      "Invalid request payload"
// @Failure      401  {object}  map[string]string      "Unauthorized or invalid OTP"
// @Failure      404  {object}  map[string]string      "Pending update not found"
// @Failure      500  {object}  map[string]string      "Internal server error"
// @Router       /api/user/me/verify-update [patch]
