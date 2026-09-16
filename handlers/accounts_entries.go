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
// @Success 200 {object} map[string]any "Journal entry updated successfully"
// @Failure 400 {object} map[string]any "Invalid request body or validation failed"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 404 {object} map[string]any "Journal entry not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/entries/{entry_id} [patch]
// @Security BearerAuth
func UpdateEntry(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", "accounts.update")
	if !ok {
		return
	}

	// Decode and validate the request body into UpdateJournalEntryRequest DTO
	req, ok := DecodeRequestBody[dtos.UpdateJournalEntryRequest](c, requestSummary, start)
	if !ok {
		return
	}

	// Validate the struct fields, ensuring debits and credits still balance
	if !utils.ValidateGinStructAndRespond(req, c, requestSummary, start, "Accounts") {
		return
	}

	// Extract entry ID from URL path parameters
	entryId := c.Param("entry_id")

	// Attempt to update the journal entry in the database
	if err := models.UpdateEntry(models.DB, entryId, *req); err != nil {
		// Return error response if update fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to update journal entry with id " + entryId,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached entries to ensure data consistency
	utils.DeleteCacheByPrefix("entries_")
	utils.DeleteCacheByPrefix("entries_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: journalWithID + entryId + " updated successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Entry updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
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
// @Success 200 {object} map[string]any "Journal entry deleted successfully"
// @Failure 401 {object} map[string]any "Unauthorized - admin access required"
// @Failure 404 {object} map[string]any "Journal entry not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/admin/entries/{entry_id} [delete]
// @Security BearerAuth
func DeleteEntry(c *gin.Context) {
	// Track request execution time
	start := time.Now()

	// Extract request summary for logging
	requestSummary := utils.GetRequestSummary(c.Request)

	// Verify that the requesting user has admin privileges
	_, ok := utils.RequireGinPermissions(c, start, requestSummary, "Accounts", "accounts.delete")
	if !ok {
		return
	}

	// Extract entry ID from URL path parameters
	entryID := c.Param("entry_id")

	// Attempt to delete the journal entry from the database
	if err := models.DeleteEntry(models.DB, entryID); err != nil {
		// Return error response if deletion fails
		utils.RespondWithGinError(c, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Accounts",
				Description: "Failed to delete journal entry with id " + entryID,
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   c.Request,
			RawBody:   requestSummary})
		return
	}

	// Invalidate cached entries to ensure data consistency
	utils.DeleteCacheByPrefix("entries_")
	utils.DeleteCacheByPrefix("entries_pagination_")
	utils.RespondWithGinJSON(c, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Accounts",
			Description: journalWithID + entryID + " deleted successfully",
			Code:        http.StatusOK,
		},
		Payload:   nil,
		Message:   "Entry deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   c.Request,
		RawBody:   requestSummary})
}
