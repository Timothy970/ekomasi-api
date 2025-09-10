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

// Create an account
// @Summary Create charts of account
// @Description Create charts of account
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/accounts [post]
func CreateAccount(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateAccountRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	_, err := models.CreateAccount(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("entries_")
	utils.DeleteCacheByPrefix("entries_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Account created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// List accounts
// @Summary List charts of account
// @Description List charts of account
// @Tags Accounts
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/accounts [get]
func ListAccounts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyAccounts := fmt.Sprintf("accounts_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("accounts_pagination_%d_size_%d", page, size)
	var accounts []dtos.ChartOfAccount
	var cachedAccounts []dtos.ChartOfAccount
	var meta dtos.PaginationMeta
	var cachedPagination dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyAccounts, &cachedAccounts)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedAccounts == nil {
		var err error
		accounts, meta, err = models.ListAccounts(page, size)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKeyAccounts, cachedAccounts)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		accounts = cachedAccounts
		meta = cachedPagination
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
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

// Get account by ID
// @Summary List charts of account by ID
// @Description List charts of account by ID
// @Tags Accounts
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/accounts/{account_id} [get]
func GetAccount(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	accountId := mux.Vars(r)["account_id"]
	acc, err := models.GetAccount(accountId)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   acc,
		Message:   "Accounts fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Update charts of account by ID
// @Description Update charts of account by ID
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/accounts/{account_id} [patch]
func UpdateAccount(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateAccountRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	accountId := mux.Vars(r)["account_id"]
	if err := models.UpdateAccount(accountId, *req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("accounts_")
	utils.DeleteCacheByPrefix("accounts_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Account updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Delete charts of account by ID
// @Description Delete charts of account by ID
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/accounts/{account_id} [delete]
func DeleteAccount(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	accountId := mux.Vars(r)["account_id"]
	if err := models.DeleteAccount(accountId); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("accounts_")
	utils.DeleteCacheByPrefix("accounts_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Account deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// Journal Entry Handlers
//
// @Summary Create journal entry
// @Description Create journal entry
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/entries [post]
func CreateEntry(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CreateJournalEntryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}

	_, err := models.CreateEntry(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("accounts_")
	utils.DeleteCacheByPrefix("accounts_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Entry created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary List journal entries
// @Description List journal entries
// @Tags Accounts
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/entries [get]
func ListEntries(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	// Ensure user is admin
	if _, ok := utils.RequireAdmin(r, w, start, requestSummary); !ok {
		return
	}
	page, size := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("size"))
	cacheKeyEntries := fmt.Sprintf("entries_%d_size_%d", page, size)
	cacheKeyPagination := fmt.Sprintf("entries_pagination_%d_size_%d", page, size)
	var entries []dtos.JournalEntry
	var cachedEntries []dtos.JournalEntry
	var meta dtos.PaginationMeta
	var cachedPagination dtos.PaginationMeta
	_ = utils.GetCache(cacheKeyEntries, &cachedEntries)
	_ = utils.GetCache(cacheKeyPagination, &cachedPagination)
	if cachedEntries == nil {
		var err error
		entries, meta, err = models.ListEntries(page, size)
		if err != nil {
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				Code:      http.StatusInternalServerError,
				Message:   err.Error(),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary})
			return
		}
		_ = utils.SetCache(cacheKeyEntries, cachedEntries)
		_ = utils.SetCache(cacheKeyPagination, cachedPagination)
	} else {
		entries = cachedEntries
		meta = cachedPagination
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code: http.StatusOK,
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

// @Summary List journal entry by ID
// @Description List journal entry by ID
// @Tags Accounts
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/entries/{entry_id} [get]
func GetEntry(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	entryId := mux.Vars(r)["entry_id"]
	entry, err := models.GetEntry(entryId)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   entry,
		Message:   "Entry fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Update journal entry by ID
// @Description Update journal entry by ID
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/entries/{entry_id} [patch]
func UpdateEntry(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.UpdateJournalEntryRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	entryId := mux.Vars(r)["entry_id"]
	if err := models.UpdateEntry(entryId, *req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("entries_")
	utils.DeleteCacheByPrefix("entries_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Entry updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Delete journal entry by ID
// @Description Delete journal entry by ID
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/entries/{entry_id} [delete]
func DeleteEntry(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	entryID := mux.Vars(r)["entry_id"]
	if err := models.DeleteEntry(entryID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.DeleteCacheByPrefix("entries_")
	utils.DeleteCacheByPrefix("entries_pagination_")
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusOK,
		Payload:   nil,
		Message:   "Entry fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
