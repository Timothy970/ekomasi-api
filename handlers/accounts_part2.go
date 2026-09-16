// Package handlers provides HTTP request handlers for the Ekomasi backend API.
// This file contains handlers for managing chart of accounts and journal entries,
// which form the core of the accounting/financial management system.
package handlers

import (
	"ekomasi_backend/dtos"
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// helper function to validate account type
// param accountType is the type of account being validated
// returns true if the account type is valid, false otherwise
func IsValidAccountType(accountType string) bool {
	validTypes := []string{"asset", "liability", "equity", "revenue", "expense"}
	for _, v := range validTypes {
		log.Printf("checking this account type %s", strings.ToLower(accountType))
		if strings.ToLower(accountType) == v {
			return true
		}
	}
	return false
}

// helper function to validate statement type and its compatibility with account type
// param statementType is the type of financial statement the account belongs to
// returns an error if the statement type is invalid or incompatible with account type
func IsValidStatementType(statementType string, accountType string) error {
	//first check if statement type is valid
	validTypes := []string{"balance sheet", "income statement", "cash flow statement"}
	statementTypeLower := strings.ToLower(statementType)
	accountTypeLower := strings.ToLower(accountType)

	isValidType := false
	for _, v := range validTypes {
		if statementTypeLower == v {
			isValidType = true
			break
		}
	}

	if !isValidType {
		return fmt.Errorf("Statement type must be one of: balance sheet, income statement, cash flow statement")
	}

	//then check compatibility with account type
	// Assets, Liabilities, Equity → Balance Sheet
	// Revenue, Expenses → Income Statement
	if (accountTypeLower == "asset" || accountTypeLower == "liability" || accountTypeLower == "equity") && statementTypeLower != "balance sheet" {
		return fmt.Errorf("Account type '%s' must belong to Balance Sheet statement", accountType)
	}

	if (accountTypeLower == "revenue" || accountTypeLower == "expense") && statementTypeLower != "income statement" {
		return fmt.Errorf("Account type '%s' must belong to Income Statement", accountType)
	}

	return nil
}

// GetAccountStats retrieves statistics about the chart of accounts.
// This endpoint provides an overview of account counts and balances.
//
// @Summary Get chart of accounts statistics
// @Description Returns total active/archived accounts and grouped stats by account type
// @Tags Accounts
// @Produce json
// @Success 200 {object} map[string]interface{} "Statistics retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/accounts/stats [get]
// @Security BearerAuth
func GetAccountStats(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user has required permissions
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", ""); !ok {
		return
	}

	// Fetch statistics from model
	stats, err := models.GetAccountStats(models.DB)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to fetch account statistics",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: "Account statistics fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   stats,
		Message:   "Account statistics fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListAccounts retrieves a paginated list of all chart of accounts.
// This endpoint requires admin privileges and implements caching for performance.
// Results are cached based on page number and page size parameters.
//
// @Summary List all chart of accounts with pagination
// @Description Retrieves a paginated list of all accounts in the chart of accounts
// @Tags Accounts
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param size query int false "Page size (default: 10)"
// @Success 200 {object} map[string]interface{} "Accounts retrieved successfully with pagination metadata"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/accounts [get]
// @Security BearerAuth
func ListAccounts(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Ensure user has admin privileges
	if _, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", ""); !ok {
		return
	}

	// Parse pagination parameters from query string
	page, size := parsePagination(c.Query("page"), c.Query("size"))
	accountType := c.Query("account_type")
	q := c.Query("q")
	// Fetch accounts from database with pagination
	accounts, meta, err := models.ListAccounts(models.DB, page, size, accountType, q)
	if err != nil {
		// Return error response if database query fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to list accounts",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: "All Accounts fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"accounts":   accounts,
			"pagination": meta,
		},
		Message:   "Accounts fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetAccount retrieves a single account by its unique identifier.
// This endpoint returns detailed information about a specific account
// from the chart of accounts.
//
// @Summary Get a chart of account by ID
// @Description Retrieves detailed information about a specific account by its unique identifier
// @Tags Accounts
// @Produce json
// @Param account_id path string true "Account ID"
// @Success 200 {object} map[string]interface{} "Account details retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/accounts/{account_id} [get]
// @Security BearerAuth
func GetAccount(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract account ID from URL path parameters
	accountId := c.Param("account_id")

	// Fetch account from database by ID
	acc, err := models.GetAccount(models.DB, accountId)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to fetch account with id " + accountId,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: accountWithID + accountId + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   acc,
		Message:   "Accounts fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// UpdateAccount modifies an existing account in the chart of accounts.
// This endpoint is restricted to admin users and allows updating account details.
// Upon successful update, related cache entries are invalidated.
//
// @Summary Update a chart of account by ID
// @Description Updates an existing account's information in the chart of accounts
// @Tags Admin
// @Accept json
// @Produce json
// @Param account_id path string true "Account ID"
// @Param request body dtos.UpdateAccountRequest true "Account update request"
// @Success 200 {object} map[string]interface{} "Account updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/accounts/{account_id} [patch]
// @Security BearerAuth
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
