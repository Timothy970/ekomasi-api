// Package handlers provides HTTP request handlers for order returns and refund management.
// This file contains handlers for processing customer return requests, managing return status
// workflows (pending, approved, rejected, completed), and tracking return items. The return
// system helps handle product returns, quality issues, and customer satisfaction.
package handlers

import (
	"ekomasi_backend/middleware"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetOwnerReturnsHandler retrieves details of a specific return request for the authenticated user.
// Allows customers to view their own return details. Access is restricted to the return owner
// for security - users cannot view other users' returns.
//
// @Summary      Get user's return request details
// @Description  Retrieve detailed information about a specific return request owned by the user
// @Tags         Returns
// @Produce      json
// @Param        return_id  path      string                true  "Return ID"
// @Success      200        {object}  dtos.ReturnResponse   "Return details"
// @Failure      400        {object}  dtos.ErrorResponse    "Return not found or not owned by user"
// @Failure      401        {object}  dtos.ErrorResponse    "User not authenticated"
// @Security     BearerAuth
// @Router       /api/my-returns/{return_id} [get]
func GetOwnerReturnsHandler(c *gin.Context) {
	// Start performance tracking for this request
	start := time.Now()
	// Get request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)
	// Extract authenticated user from request context
	authuser, ok := middleware.UserFromContext(c.Request.Context())
	if !ok {
		// User not authenticated, return unauthorized error
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Users",
				Description: "User not found in context or not authenticated",
				Code:        http.StatusUnauthorized,
			},
			Message:   "User not validated",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	// Extract return ID from URL path parameters
	returnID := c.Param("return_id")
	// Fetch return details from database (verifies user ownership)
	ret, err := models.GetOwnerReturnByID(models.DB, returnID, authuser.ID)
	if err != nil {
		// Return not found or not owned by user
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Orders",
				Description: "Failed to fetch return details " + err.Error(),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	// Return success response with return details for user's own return
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Orders",
			Description: "Return details fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   ret,
		Message:   "Return details fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
