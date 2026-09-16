// Package handlers provides HTTP request handlers for the Ekomasi backend API.
// This file contains handlers for managing chart of accounts and journal entries,
// which form the core of the accounting/financial management system.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Global message template variables for consistent response formatting
var (
	// accountWithID is a message prefix for account-related operations
	accountWithID = "Account with id "
	// journalWithID is a message prefix for journal entry-related operations
	journalWithID = "Journal entry with id "
)

// CreateAccount handles the creation of a new chart of account entry.
// This endpoint is restricted to admin users only and creates an account
// that can be used for financial tracking and journal entries.
//
// @Summary Create a new chart of account
// @Description Creates a new account in the chart of accounts for financial tracking
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dtos.CreateAccountRequest true "Account creation request"
// @Success 201 {object} map[string]interface{} "Account created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/accounts [post]
// @Security BearerAuth
func CreateAccount(c *gin.Context) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", "accounts.create")
	if !ok {
		return
	}

	// Decode and validate the request body into CreateAccountRequest DTO
	req, ok := DecodeRequestBody[dtos.CreateAccountRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Accounts") {
		return
	}
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
	var accountCode string
	codeRange, err := utils.GetAccountCodeRange(req.AccountType)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to get account code range for account type " + req.AccountType,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}
	// Auto-generate account code if not provided
	if req.AccountCode == nil || *req.AccountCode == "" {
		accountCode, err = models.GetNextAccountCode(models.DB, req.AccountType, codeRange)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Accounts",
					Description: "Failed to generate account code for account type " + req.AccountType,
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   c.Request,
				RawBody:   requestSummary,
			})
			return
		}

	} else {
		accountCode = *req.AccountCode

		// Validate that the provided code is within the valid range for this account type
		err = utils.ValidateAccountCode(accountCode, req.AccountType)
		if err != nil {
			utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Accounts",
					Description: "Invalid account code provided: " + err.Error(),
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

	}

	// Attempt to create the account in the database
	_, err = models.CreateAccount(models.DB, *req, accountCode)
	if err != nil {
		// Return error response if account creation fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to create account",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate related cache entries to ensure data consistency
	utils.DeleteCacheByPrefix("entries_")
	utils.DeleteCacheByPrefix("entries_pagination_")

	// Return success response with account creation confirmation
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: "Account created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Account created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetNextAccountCode retrieves the next available account code for a given account type.
// This endpoint helps the UI display the next available code before creating an account.
// Each account type has a dedicated range (e.g., Assets: 1000-1999, Liabilities: 2000-2999).
//
// @Summary Get next available account code by account type
// @Description Returns the next available account code for a specific account type with range information
// @Tags Accounts
// @Produce json
// @Param account_type query string true "Account Type (asset, liability, equity, revenue, expense)"
// @Success 200 {object} map[string]interface{} "Next available code retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid account type"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/accounts/next-code [get]
// @Security BearerAuth
func GetNextAccountCode(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user has required permissions
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", ""); !ok {
		return
	}

	// Get account type from query parameter
	accountType := c.Query("account_type")
	if accountType == "" {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Account type is required",
				Code:        http.StatusBadRequest,
			},
			Message:   "account_type query parameter is required",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Validate account type and get code range
	codeRange, err := utils.GetAccountCodeRange(accountType)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Invalid account type",
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

	// Get next available code from database
	nextCode, err := models.GetNextAccountCode(models.DB, accountType, codeRange)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to get next account code",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary,
		})
		return
	}

	// Build response
	response := dtos.NextAccountCodeResponse{
		AccountType: accountType,
		NextCode:    nextCode,
		MinCode:     codeRange.Min,
		MaxCode:     codeRange.Max,
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: "Next account code retrieved successfully",
			Code:        http.StatusOK,
		},
		Payload:   response,
		Message:   "Next account code retrieved successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary,
	})
}
