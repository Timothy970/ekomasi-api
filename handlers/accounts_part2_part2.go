package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func UpdateAccount(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", "accounts.update")
	if !ok {
		return
	}

	// Decode and validate the request body into UpdateAccountRequest DTO
	req, ok := DecodeRequestBody[dtos.UpdateAccountRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Accounts") {
		return
	}

	// Extract account ID from URL path parameters
	accountId := c.Param("account_id")

	//check that account type is valid
	if !IsValidAccountType(req.AccountType) {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Invalid account type provided",
				Code:        http.StatusBadRequest,
			},
			Message:   "Account type must be one of: asset, liability, equity, revenue, expense",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	//check if statement type is valid and if it matches the account type
	err := IsValidStatementType(req.StatementType, req.AccountType)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Invalid statement type provided " + err.Error(),
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
	// Attempt to update the account in the database
	if err := models.UpdateAccount(models.DB, accountId, *req); err != nil {
		// Return error response if update fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to update account with id " + accountId,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached accounts to ensure data consistency
	utils.DeleteCacheByPrefix("accounts_")
	utils.DeleteCacheByPrefix("accounts_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: accountWithID + accountId + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Account updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
