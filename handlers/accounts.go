// Package handlers provides HTTP request handlers for the Adenzo backend API.
// This file contains handlers for managing chart of accounts and journal entries,
// which form the core of the accounting/financial management system.
package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
func CreateAccount(w http.ResponseWriter, r *http.Request) {
	// Track request execution time for performance monitoring
	start := time.Now()

	// Extract request summary for logging and error reporting
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Accounts", "accounts.create")
	if !ok {
		return
	}

	// Decode and validate the request body into CreateAccountRequest DTO
	req, ok := DecodeRequestBody[dtos.CreateAccountRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Accounts") {
		return
	}

	// Attempt to create the account in the database
	_, err := models.CreateAccount(*req)
	if err != nil {
		// Return error response if account creation fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to create account",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate related cache entries to ensure data consistency
	utils.DeleteCacheByPrefix("entries_")
	utils.DeleteCacheByPrefix("entries_pagination_")

	// Return success response with account creation confirmation
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: "Account created successfully",
			Code:        http.StatusCreated,
		},
		Payload:   nil,
		Message:   "Account created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func ListAccounts(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Accounts", ""); !ok {
		return
	}

	// Parse pagination parameters from query string
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	// Generate cache keys for both accounts data and pagination metadata
	cacheKeyAccounts := fmt.Sprintf("accounts_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("accounts_pagination_%d_size_%d", page, size)

	// Initialize variables for accounts data and cached versions
	var accounts []dtos.ChartOfAccount
	var cachedAccounts []dtos.ChartOfAccount
	var meta dtos.PaginationMeta
	var cachedPagination dtos.PaginationMeta

	// Attempt to retrieve cached data
	_ = utils.GetCache(cacheKeyAccounts, &cachedAccounts)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)

	// If cache miss, fetch from database
	if cachedAccounts == nil {
		var err error
		// Fetch accounts from database with pagination
		accounts, meta, err = models.ListAccounts(page, size)
		if err != nil {
			// Return error response if database query fails
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Accounts",
					Description: "Failed to list accounts",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Cache the fetched data for future requests
		_ = utils.SetCache(cacheKeyAccounts, cachedAccounts)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		// Use cached data if available
		accounts = cachedAccounts
		meta = cachedPagination
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
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
		Request:   r,
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
func GetAccount(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract account ID from URL path parameters
	accountId := mux.Vars(r)["account_id"]

	// Fetch account from database by ID
	acc, err := models.GetAccount(accountId)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to fetch account with id " + accountId,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: accountWithID + accountId + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   acc,
		Message:   "Accounts fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
func UpdateAccount(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Accounts", "accounts.update")
	if !ok {
		return
	}

	// Decode and validate the request body into UpdateAccountRequest DTO
	req, ok := DecodeRequestBody[dtos.UpdateAccountRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields according to validation tags
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Accounts") {
		return
	}

	// Extract account ID from URL path parameters
	accountId := mux.Vars(r)["account_id"]

	// Attempt to update the account in the database
	if err := models.UpdateAccount(accountId, *req); err != nil {
		// Return error response if update fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to update account with id " + accountId,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached accounts to ensure data consistency
	utils.DeleteCacheByPrefix("accounts_")
	utils.DeleteCacheByPrefix("accounts_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: accountWithID + accountId + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Account updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteAccount removes an account from the chart of accounts.
// This endpoint is restricted to users with the "accounts.delete" permission and permanently deletes the specified account.
// Related cache entries are invalidated upon successful deletion.
//
// @Summary Delete a chart of account by ID
// @Description Permanently deletes an account from the chart of accounts
// @Tags Admin
// @Produce json
// @Param account_id path string true "Account ID"
// @Success 200 {object} map[string]interface{} "Account deleted successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Account not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/accounts/{account_id} [delete]
// @Security BearerAuth
func DeleteAccount(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Accounts", "accounts.delete")
	if !ok {
		return
	}

	// Extract account ID from URL path parameters
	accountId := mux.Vars(r)["account_id"]

	// Attempt to delete the account from the database
	if err := models.DeleteAccount(accountId); err != nil {
		// Return error response if deletion fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to delete account with id " + accountId,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached accounts to ensure data consistency
	utils.DeleteCacheByPrefix("accounts_")
	utils.DeleteCacheByPrefix("accounts_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: accountWithID + accountId + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Account deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success 200 {object} map[string]interface{} "Journal entry created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/entries [post]
// @Security BearerAuth
func CreateEntry(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Accounts", "accounts.create")
	if !ok {
		return
	}

	// Decode and validate the request body into CreateJournalEntryRequest DTO
	req, ok := DecodeRequestBody[dtos.CreateJournalEntryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields, ensuring debits and credits balance
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Accounts") {
		return
	}

	// Attempt to create the journal entry in the database
	_, err := models.CreateEntry(*req)
	if err != nil {
		// Return error response if entry creation fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to create journal entry",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate account cache as journal entries affect account balances
	utils.DeleteCacheByPrefix("accounts_")
	utils.DeleteCacheByPrefix("accounts_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: "Entry created successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Entry created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success 200 {object} map[string]interface{} "Journal entries retrieved successfully with pagination metadata"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/entries [get]
// @Security BearerAuth
func ListEntries(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Ensure user has admin privileges
	if _, ok := utils.RequirePermissions(r, w, start, requestSummary, "Accounts", ""); !ok {
		return
	}

	// Parse pagination parameters from query string
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	// Generate cache keys for both entries data and pagination metadata
	cacheKeyEntries := fmt.Sprintf("entries_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("entries_pagination_%d_size_%d", page, size)

	// Initialize variables for entries data and cached versions
	var entries []dtos.JournalEntry
	var cachedEntries []dtos.JournalEntry
	var meta dtos.PaginationMeta
	var cachedPagination dtos.PaginationMeta

	// Attempt to retrieve cached data
	_ = utils.GetCache(cacheKeyEntries, &cachedEntries)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)

	// If cache miss, fetch from database
	if cachedEntries == nil {
		var err error
		// Fetch journal entries from database with pagination
		entries, meta, err = models.ListEntries(page, size)
		if err != nil {
			// Return error response if database query fails
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Accounts",
					Description: "Failed to list journal entries",
					Code:        http.StatusInternalServerError,
				},
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		// Cache the fetched data for future requests
		_ = utils.SetCache(cacheKeyEntries, cachedEntries)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		// Use cached data if available
		entries = cachedEntries
		meta = cachedPagination
	}

	// Return success response with entries and pagination metadata
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: "Journal entries fetched successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]interface{}{
			"accounts":   entries,
			"pagination": meta,
		},
		Message:   "Entries fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
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
// @Success 200 {object} map[string]interface{} "Journal entry details retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Journal entry not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/entries/{entry_id} [get]
// @Security BearerAuth
func GetEntry(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Extract entry ID from URL path parameters
	entryId := mux.Vars(r)["entry_id"]

	// Fetch journal entry from database by ID
	entry, err := models.GetEntry(entryId)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to fetch journal entry with id " + entryId,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: journalWithID + entryId + " fetched successfully",
			Code:        http.StatusOK,
		},
		Payload:   entry,
		Message:   "Entry fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// UpdateEntry modifies an existing journal entry in the accounting system.
// This endpoint is restricted to admin users and allows updating entry details
// while maintaining the integrity of the double-entry bookkeeping system.
//
// @Summary Update a journal entry by ID
// @Description Updates an existing journal entry's information
// @Tags Admin
// @Accept json
// @Produce json
// @Param entry_id path string true "Journal Entry ID"
// @Param request body dtos.UpdateJournalEntryRequest true "Journal entry update request"
// @Success 200 {object} map[string]interface{} "Journal entry updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Journal entry not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/entries/{entry_id} [patch]
// @Security BearerAuth
func UpdateEntry(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Accounts", "accounts.update")
	if !ok {
		return
	}

	// Decode and validate the request body into UpdateJournalEntryRequest DTO
	req, ok := DecodeRequestBody[dtos.UpdateJournalEntryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields, ensuring debits and credits still balance
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Accounts") {
		return
	}

	// Extract entry ID from URL path parameters
	entryId := mux.Vars(r)["entry_id"]

	// Attempt to update the journal entry in the database
	if err := models.UpdateEntry(entryId, *req); err != nil {
		// Return error response if update fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to update journal entry with id " + entryId,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached entries to ensure data consistency
	utils.DeleteCacheByPrefix("entries_")
	utils.DeleteCacheByPrefix("entries_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: journalWithID + entryId + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Entry updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// DeleteEntry removes a journal entry from the accounting system.
// This endpoint is restricted to admin users and permanently deletes the specified
// journal entry. Use with caution as this affects the financial records.
//
// @Summary Delete a journal entry by ID
// @Description Permanently deletes a journal entry from the accounting system
// @Tags Admin
// @Produce json
// @Param entry_id path string true "Journal Entry ID"
// @Success 200 {object} map[string]interface{} "Journal entry deleted successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - admin access required"
// @Failure 404 {object} map[string]interface{} "Journal entry not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/admin/entries/{entry_id} [delete]
// @Security BearerAuth
func DeleteEntry(w http.ResponseWriter, r *http.Request) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(r)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequirePermissions(r, w, start, requestSummary, "Accounts", "accounts.delete")
	if !ok {
		return
	}

	// Extract entry ID from URL path parameters
	entryID := mux.Vars(r)["entry_id"]

	// Attempt to delete the journal entry from the database
	if err := models.DeleteEntry(entryID); err != nil {
		// Return error response if deletion fails
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to delete journal entry with id " + entryID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached entries to ensure data consistency
	utils.DeleteCacheByPrefix("entries_")
	utils.DeleteCacheByPrefix("entries_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: journalWithID + entryID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Entry deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
