package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"adenzo_backend/dtos"
	"adenzo_backend/middleware"
	"adenzo_backend/models"
	"adenzo_backend/utils"

	"github.com/gorilla/mux"
)

// --- Mocks and helpers ---
func mockRequireAdminPass(r *http.Request, w http.ResponseWriter, start time.Time, requestSummary string) (middleware.AuthenticatedUser, bool) {
	return middleware.AuthenticatedUser{Role: "admin"}, true
}
func mockRequireAdminFail(r *http.Request, w http.ResponseWriter, start time.Time, requestSummary string) (middleware.AuthenticatedUser, bool) {
	return middleware.AuthenticatedUser{}, false
}
func mockGetRequestSummary(r *http.Request) string { return "" }
func mockValidateStructAndRespond(req any, w http.ResponseWriter, r *http.Request, reqSummary string, start time.Time) bool {
	return true
}
func mockRespondWithJSON(w http.ResponseWriter, opts utils.SuccessJSONResponseOptions) {
	w.WriteHeader(opts.Code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": opts.Message,
		"payload": opts.Payload,
	})
}
func mockRespondWithError(w http.ResponseWriter, opts utils.ErrorJSONResponseOptions) {
	w.WriteHeader(opts.Code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": opts.Message,
	})
}
func mockDeleteCacheByPrefix(prefix string) error                                       { return nil }
func mockGetCache(key string, dest interface{}) error                                   { return nil }
func mockSetCache(key string, val interface{}, customExpiration ...time.Duration) error { return nil }

func patchHandlersForTest(adminOk bool) {
	if adminOk {
		utils.RequireAdmin = mockRequireAdminPass
	} else {
		utils.RequireAdmin = mockRequireAdminFail
	}
	utils.GetRequestSummary = mockGetRequestSummary
	utils.ValidateStructAndRespond = mockValidateStructAndRespond
	utils.RespondWithJSON = mockRespondWithJSON
	utils.RespondWithError = mockRespondWithError
	utils.DeleteCacheByPrefix = mockDeleteCacheByPrefix
	utils.GetCache = mockGetCache
	utils.SetCache = mockSetCache
}

// --- DecodeRequestBody mocks for each type ---
var (
	DecodeRequestBodyCreateAccountRequest      func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.CreateAccountRequest, bool)
	DecodeRequestBodyUpdateAccountRequest      func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.UpdateAccountRequest, bool)
	DecodeRequestBodyCreateJournalEntryRequest func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.CreateJournalEntryRequest, bool)
	DecodeRequestBodyUpdateJournalEntryRequest func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.UpdateJournalEntryRequest, bool)
)

func patchDecodeRequestBody() {
	DecodeRequestBodyCreateAccountRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.CreateAccountRequest, bool) {
		var t dtos.CreateAccountRequest
		_ = json.NewDecoder(r.Body).Decode(&t)
		return &t, true
	}
	DecodeRequestBodyUpdateAccountRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.UpdateAccountRequest, bool) {
		var t dtos.UpdateAccountRequest
		_ = json.NewDecoder(r.Body).Decode(&t)
		return &t, true
	}
	DecodeRequestBodyCreateJournalEntryRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.CreateJournalEntryRequest, bool) {
		var t dtos.CreateJournalEntryRequest
		_ = json.NewDecoder(r.Body).Decode(&t)
		return &t, true
	}
	DecodeRequestBodyUpdateJournalEntryRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.UpdateJournalEntryRequest, bool) {
		var t dtos.UpdateJournalEntryRequest
		_ = json.NewDecoder(r.Body).Decode(&t)
		return &t, true
	}
}

// --- Models mocks ---
const (
	createErrorMsg = "create error"
	listErrorMsg   = "list error"
	notFoundMsg    = "not found"
	updateErrorMsg = "update error"
	deleteErrorMsg = "delete error"
)

func patchModelsForTest() {
	// Adjust the field name below to match your actual dtos.CreateAccountRequest struct, e.g. req.AccountName or req.Name
	models.CreateAccount = func(req dtos.CreateAccountRequest) (string, error) {
		if req.AccountName == "fail" {
			return "", errors.New(createErrorMsg)
		}
		return "mockID", nil
	}
	models.ListAccounts = func(page, size int) ([]dtos.ChartOfAccount, dtos.PaginationMeta, error) {
		if page == 99 {
			return nil, dtos.PaginationMeta{}, errors.New(listErrorMsg)
		}
		return []dtos.ChartOfAccount{{AccountID: "1", AccountName: "Cash"}}, dtos.PaginationMeta{TotalPages: 1}, nil
	}
	models.GetAccount = func(id string) (dtos.ChartOfAccount, error) {
		if id == "notfound" {
			return dtos.ChartOfAccount{}, errors.New(notFoundMsg)
		}
		return dtos.ChartOfAccount{AccountID: id, AccountName: "Cash"}, nil
	}
	models.UpdateAccount = func(id string, req dtos.UpdateAccountRequest) error {
		if req.AccountName == "fail" {
			return errors.New(updateErrorMsg)
		}
		return nil
	}
	models.DeleteAccount = func(id string) error {
		if id == "notfound" {
			return errors.New(deleteErrorMsg)
		}
		return nil
	}
	models.CreateEntry = func(req dtos.CreateJournalEntryRequest) (string, error) {
		if req.Description == nil {
			return "", errors.New(createErrorMsg)
		}
		return "mockEntryID", nil
	}
	models.ListEntries = func(page, size int) ([]dtos.JournalEntry, dtos.PaginationMeta, error) {
		if page == 99 {
			return nil, dtos.PaginationMeta{}, errors.New(listErrorMsg)
		}
		desc := "Test"
		return []dtos.JournalEntry{{EntryID: "1", Description: &desc}}, dtos.PaginationMeta{TotalPages: 1}, nil
	}
	models.GetEntry = func(id string) (dtos.JournalEntry, error) {
		if id == "notfound" {
			return dtos.JournalEntry{}, errors.New(notFoundMsg)
		}
		desc := "Test"
		return dtos.JournalEntry{EntryID: id, Description: &desc}, nil
	}
	models.UpdateEntry = func(id string, req dtos.UpdateJournalEntryRequest) error {
		if req.Debit == 0 && req.Credit == 0 {
			return errors.New(updateErrorMsg)
		}
		return nil
	}
	models.DeleteEntry = func(id string) error {
		if id == "notfound" {
			return errors.New(deleteErrorMsg)
		}
		return nil
	}
}

// --- Table Driven Tests ---

func TestCreateAccountHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name     string
		adminOk  bool
		input    dtos.CreateAccountRequest
		wantCode int
		wantMsg  string
	}{
		// Expected: 201, message "Account created successfully"
		{"success", true, dtos.CreateAccountRequest{AccountName: "Cash"}, http.StatusCreated, "Account created successfully"},
		// Expected: 500, message "create error"
		{"model error", true, dtos.CreateAccountRequest{AccountName: "fail"}, http.StatusInternalServerError, "create error"},
		// Expected: 0, no response (admin fail)
		{"admin fail", false, dtos.CreateAccountRequest{AccountName: "Cash"}, 0, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(tc.adminOk)
			body, _ := json.Marshal(tc.input)
			req := httptest.NewRequest("POST", "/api/admin/accounts", bytes.NewReader(body))
			w := httptest.NewRecorder()
			// Use patched function
			DecodeRequestBodyCreateAccountRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.CreateAccountRequest, bool) {
				var t dtos.CreateAccountRequest
				_ = json.NewDecoder(r.Body).Decode(&t)
				return &t, true
			}
			CreateAccount(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestListAccountsHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name     string
		adminOk  bool
		page     string
		wantCode int
		wantMsg  string
	}{
		// Expected: 200, message "Accounts fetched successfully"
		{"success", true, "1", http.StatusOK, "Accounts fetched successfully"},
		// Expected: 500, message "list error"
		{"model error", true, "99", http.StatusInternalServerError, "list error"},
		// Expected: 0, no response (admin fail)
		{"admin fail", false, "1", 0, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(tc.adminOk)
			req := httptest.NewRequest("GET", "/api/accounts?page="+tc.page+"&size=10", nil)
			w := httptest.NewRecorder()
			ListAccounts(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestGetAccountHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name      string
		accountID string
		wantCode  int
		wantMsg   string
	}{
		// Expected: 200, message "Accounts fetched successfully"
		{"success", "1", http.StatusOK, "Accounts fetched successfully"},
		// Expected: 500, message "not found"
		{"not found", "notfound", http.StatusInternalServerError, "not found"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(true)
			req := httptest.NewRequest("GET", "/api/accounts/"+tc.accountID, nil)
			req = mux.SetURLVars(req, map[string]string{"account_id": tc.accountID})
			w := httptest.NewRecorder()
			GetAccount(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestUpdateAccountHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name      string
		adminOk   bool
		accountID string
		input     dtos.UpdateAccountRequest
		wantCode  int
		wantMsg   string
	}{
		// Expected: 200, message "Account updated successfully"
		{"success", true, "1", dtos.UpdateAccountRequest{AccountName: "Cash"}, http.StatusOK, "Account updated successfully"},
		// Expected: 500, message "update error"
		{"model error", true, "1", dtos.UpdateAccountRequest{AccountName: "fail"}, http.StatusInternalServerError, "update error"},
		// Expected: 0, no response (admin fail)
		{"admin fail", false, "1", dtos.UpdateAccountRequest{AccountName: "Cash"}, 0, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(tc.adminOk)
			body, _ := json.Marshal(tc.input)
			req := httptest.NewRequest("PATCH", "/api/admin/accounts/"+tc.accountID, bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"account_id": tc.accountID})
			w := httptest.NewRecorder()
			DecodeRequestBodyUpdateAccountRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.UpdateAccountRequest, bool) {
				var t dtos.UpdateAccountRequest
				_ = json.NewDecoder(r.Body).Decode(&t)
				return &t, true
			}
			UpdateAccount(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestDeleteAccountHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name      string
		adminOk   bool
		accountID string
		wantCode  int
		wantMsg   string
	}{
		// Expected: 200, message "Account deleted successfully"
		{"success", true, "1", http.StatusOK, "Account deleted successfully"},
		// Expected: 500, message "delete error"
		{"model error", true, "notfound", http.StatusInternalServerError, "delete error"},
		// Expected: 0, no response (admin fail)
		{"admin fail", false, "1", 0, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(tc.adminOk)
			req := httptest.NewRequest("DELETE", "/api/admin/accounts/"+tc.accountID, nil)
			req = mux.SetURLVars(req, map[string]string{"account_id": tc.accountID})
			w := httptest.NewRecorder()
			DeleteAccount(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestCreateEntryHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name     string
		adminOk  bool
		input    dtos.CreateJournalEntryRequest
		wantCode int
		wantMsg  string
	}{
		// Expected: 200, message "Entry created successfully"

		{"success", true, dtos.CreateJournalEntryRequest{Description: func() *string { s := "Test"; return &s }()}, http.StatusOK, "Entry created successfully"},
		// Expected: 500, message "create error"
		{"model error", true, dtos.CreateJournalEntryRequest{Description: func() *string { s := "fail"; return &s }()}, http.StatusInternalServerError, "create error"},
		// Expected: 0, no response (admin fail)
		{"admin fail", false, dtos.CreateJournalEntryRequest{Description: func() *string { s := "Test"; return &s }()}, 0, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(tc.adminOk)
			body, _ := json.Marshal(tc.input)
			req := httptest.NewRequest("POST", "/api/admin/entries", bytes.NewReader(body))
			w := httptest.NewRecorder()
			DecodeRequestBodyCreateJournalEntryRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.CreateJournalEntryRequest, bool) {
				var t dtos.CreateJournalEntryRequest
				_ = json.NewDecoder(r.Body).Decode(&t)
				return &t, true
			}
			CreateEntry(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestListEntriesHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name     string
		adminOk  bool
		page     string
		wantCode int
		wantMsg  string
	}{
		// Expected: 200, message "Entries fetched successfully"
		{"success", true, "1", http.StatusOK, "Entries fetched successfully"},
		// Expected: 500, message "list error"
		{"model error", true, "99", http.StatusInternalServerError, "list error"},
		// Expected: 0, no response (admin fail)
		{"admin fail", false, "1", 0, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(tc.adminOk)
			req := httptest.NewRequest("GET", "/api/entries?page="+tc.page+"&size=10", nil)
			w := httptest.NewRecorder()
			ListEntries(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestGetEntryHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name     string
		entryID  string
		wantCode int
		wantMsg  string
	}{
		// Expected: 200, message "Entry fetched successfully"
		{"success", "1", http.StatusOK, "Entry fetched successfully"},
		// Expected: 500, message "not found"
		{"not found", "notfound", http.StatusInternalServerError, "not found"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(true)
			req := httptest.NewRequest("GET", "/api/entries/"+tc.entryID, nil)
			req = mux.SetURLVars(req, map[string]string{"entry_id": tc.entryID})
			w := httptest.NewRecorder()
			GetEntry(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestUpdateEntryHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name     string
		adminOk  bool
		entryID  string
		input    dtos.UpdateJournalEntryRequest
		wantCode int
		wantMsg  string
	}{
		// Expected: 200, message "Entry updated successfully"
		{"success", true, "1", dtos.UpdateJournalEntryRequest{Debit: 100, Credit: 0}, http.StatusOK, "Entry updated successfully"},
		// Expected: 500, message "update error"
		{"model error", true, "1", dtos.UpdateJournalEntryRequest{Debit: 0, Credit: 0}, http.StatusInternalServerError, "update error"},
		// Expected: 0, no response (admin fail)
		{"admin fail", false, "1", dtos.UpdateJournalEntryRequest{Debit: 100, Credit: 0}, 0, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(tc.adminOk)
			body, _ := json.Marshal(tc.input)
			req := httptest.NewRequest("PATCH", "/api/admin/entries/"+tc.entryID, bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"entry_id": tc.entryID})
			w := httptest.NewRecorder()
			DecodeRequestBodyUpdateJournalEntryRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.UpdateJournalEntryRequest, bool) {
				var t dtos.UpdateJournalEntryRequest
				_ = json.NewDecoder(r.Body).Decode(&t)
				return &t, true
			}
			UpdateEntry(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestDeleteEntryHandler(t *testing.T) {
	patchDecodeRequestBody()
	patchModelsForTest()
	testCases := []struct {
		name     string
		adminOk  bool
		entryID  string
		wantCode int
		wantMsg  string
	}{
		// Expected: 200, message "Entry fetched successfully"
		{"success", true, "1", http.StatusOK, "Entry fetched successfully"},
		// Expected: 500, message "delete error"
		{"model error", true, "notfound", http.StatusInternalServerError, "delete error"},
		// Expected: 0, no response (admin fail)
		{"admin fail", false, "1", 0, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patchHandlersForTest(tc.adminOk)
			req := httptest.NewRequest("DELETE", "/api/admin/entries/"+tc.entryID, nil)
			req = mux.SetURLVars(req, map[string]string{"entry_id": tc.entryID})
			w := httptest.NewRecorder()
			DeleteEntry(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("expected %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

// --- Benchmark Tests ---

func BenchmarkCreateAccountHandler(b *testing.B) {
	patchDecodeRequestBody()
	patchModelsForTest()
	patchHandlersForTest(true)
	body, _ := json.Marshal(dtos.CreateAccountRequest{AccountName: "Cash"})
	req := httptest.NewRequest("POST", "/api/admin/accounts", bytes.NewReader(body))
	DecodeRequestBodyCreateAccountRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.CreateAccountRequest, bool) {
		var t dtos.CreateAccountRequest
		_ = json.NewDecoder(r.Body).Decode(&t)
		return &t, true
	}
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		CreateAccount(w, req)
	}
}

func BenchmarkListAccountsHandler(b *testing.B) {
	patchDecodeRequestBody()
	patchModelsForTest()
	patchHandlersForTest(true)
	req := httptest.NewRequest("GET", "/api/accounts?page=1&size=10", nil)
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		ListAccounts(w, req)
	}
}

func BenchmarkCreateEntryHandler(b *testing.B) {
	patchDecodeRequestBody()
	patchModelsForTest()
	patchHandlersForTest(true)
	s := "Test"
	body, _ := json.Marshal(dtos.CreateJournalEntryRequest{Description: &s})
	req := httptest.NewRequest("POST", "/api/admin/entries", bytes.NewReader(body))
	DecodeRequestBodyCreateJournalEntryRequest = func(r *http.Request, w http.ResponseWriter, reqSummary interface{}, start time.Time) (*dtos.CreateJournalEntryRequest, bool) {
		var t dtos.CreateJournalEntryRequest
		_ = json.NewDecoder(r.Body).Decode(&t)
		return &t, true
	}
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		CreateEntry(w, req)
	}
}
