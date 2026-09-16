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

// DeleteAccount removes an account from the chart of accounts.
// This endpoint is restricted to users with the "accounts.delete" permission and permanently deletes the specified account.
// Related cache entries are invalidated upon successful deletion.
//
// @Summary Delete a chart of account by ID
// @Description Permanently deletes an account from the chart of accounts
// @Tags Admin
// @Produce json
// @Param account_id path string true "Account ID"
// @Success 200 {object} map[string]any "Account deleted successfully"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 404 {object} map[string]any "Account not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/accounts/{account_id} [delete]
// @Security BearerAuth
func DeleteAccount(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", "accounts.delete")
	if !ok {
		return
	}

	// Extract account ID from URL path parameters
	accountId := c.Param("account_id")

	// Attempt to delete the account from the database
	if err := models.DeleteAccount(models.DB, accountId); err != nil {
		// Return error response if deletion fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to delete account with id " + accountId,
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
			Description: accountWithID + accountId + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Account deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ============================================================================
// Journal Entry Handlers
// ============================================================================
// The following handlers manage journal entries, which record financial transactions
// against the chart of accounts. Journal entries maintain the double-entry
// bookkeeping system with debits and credits.

// CreateEntry creates a new journal entry in the accounting system.
// This endpoint is restricted to admin users and creates a financial transaction
// record with debits and credits that must balance.
//
// @Summary Create a new journal entry
// @Description Creates a new journal entry for recording financial transactions
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dtos.CreateJournalEntryRequest true "Journal entry creation request"
// @Success 200 {object} map[string]any "Journal entry created successfully"
// @Failure 400 {object} map[string]any "Invalid request body or validation failed"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/entries [post]
// @Security BearerAuth
func CreateEntry(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", "accounts.create")
	if !ok {
		return
	}

	// Decode and validate the request body into CreateJournalEntryRequest DTO
	req, ok := DecodeRequestBody[dtos.CreateJournalEntryRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields, ensuring debits and credits balance
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Accounts") {
		return
	}

	// Attempt to create the journal entry in the database
	_, err := models.CreateEntry(models.DB, *req)
	if err != nil {
		// Return error response if entry creation fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to create journal entry",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate account cache as journal entries affect account balances
	utils.DeleteCacheByPrefix("accounts_")
	utils.DeleteCacheByPrefix("accounts_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: "Entry created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Entry created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// ListEntries retrieves a paginated list of all journal entries.
// This endpoint requires admin privileges and implements caching for performance.
// Results include all journal entries with their debits, credits, and related information.
//
// @Summary List all journal entries with pagination
// @Description Retrieves a paginated list of all journal entries in the accounting system
// @Tags Accounts
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param size query int false "Page size (default: 10)"
// @Success 200 {object} map[string]any "Journal entries retrieved successfully with pagination metadata"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/entries [get]
// @Security BearerAuth
func ListEntries(c *gin.Context) {
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
	q := c.Query("q")

	// Fetch journal entries from database with pagination
	entries, meta, err := models.ListEntries(models.DB, page, size, q)
	if err != nil {
		// Return error response if database query fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to list journal entries",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Return success response with entries and pagination metadata
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: "Journal entries fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"accounts":   entries,
			"pagination": meta,
		},
		Message:   "Entries fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}

// GetEntry retrieves a single journal entry by its unique identifier.
// This endpoint returns detailed information about a specific journal entry
// including all debits, credits, and related transaction details.
//
// @Summary Get a journal entry by ID
// @Description Retrieves detailed information about a specific journal entry by its unique identifier
// @Tags Accounts
// @Produce json
// @Param entry_id path string true "Journal Entry ID"
// @Success 200 {object} map[string]any "Journal entry details retrieved successfully"
// @Failure 404 {object} map[string]any "Journal entry not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/entries/{entry_id} [get]
// @Security BearerAuth
func GetEntry(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Extract entry ID from URL path parameters
	entryId := c.Param("entry_id")

	// Fetch journal entry from database by ID
	entry, err := models.GetEntry(models.DB, entryId)
	if err != nil {
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to fetch journal entry with id " + entryId,
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
			Description: journalWithID + entryId + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   entry,
		Message:   "Entry fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
