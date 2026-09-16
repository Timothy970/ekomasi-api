package handlers

import (
	"ekomasi_backend/models"
	"ekomasi_backend/utils"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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
// @Success 200 {object} map[string]any "Statistics retrieved successfully"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
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
// @Success 200 {object} map[string]any "Accounts retrieved successfully with pagination metadata"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 500 {object} map[string]any "Internal server error"
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
		Payload: map[string]any{
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
// @Success 200 {object} map[string]any "Account details retrieved successfully"
// @Failure 404 {object} map[string]any "Account not found"
// @Failure 500 {object} map[string]any "Internal server error"
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
// @Success 200 {object} map[string]any "Account updated successfully"
// @Failure 400 {object} map[string]any "Invalid request body or validation failed"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 404 {object} map[string]any "Account not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/accounts/{account_id} [patch]
// @Security BearerAuth
