package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

var invalidfrom = "invalid from date"
var invalidto = "invalid to date"
var date = "2006-01-02"

// ===== Balance Sheet =====
// GET /reports/balance-sheet?as_of={current_date}
func BalanceSheet(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Reports")
	if !ok {
		return
	}
	asOfStr := r.URL.Query().Get("as_of")
	asOf, err := time.Parse(date, asOfStr)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid as of date for balance sheet report",
				Code:        http.StatusBadRequest,
			},
			Message:   "Invalid as of date",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	assets, liabs, equity, err := models.BalanceSheet(asOf)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate balance sheet report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	toSection := func(rows []dtos.BsRow) dtos.BalanceSheetSection {
		out := dtos.BalanceSheetSection{}
		var total float64
		for _, r := range rows {
			out.Accounts = append(out.Accounts, dtos.BalanceSheetAccount{
				AccountID:   r.AccountID,
				AccountCode: r.AccountCode,
				AccountName: r.AccountName,
				Balance:     r.Balance,
			})
			total += r.Balance
		}
		out.Total = total
		return out
	}

	resp := dtos.BalanceSheetResponse{
		AsOf:        asOf,
		Assets:      toSection(assets),
		Liabilities: toSection(liabs),
		Equity:      toSection(equity),
	}
	resp.BalanceCheck = resp.Assets.Total - (resp.Liabilities.Total + resp.Equity.Total)
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: fmt.Sprintf("Balance sheet as of %s generated successfully", asOfStr),
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   fmt.Sprintf("Balance sheet as of %s", asOfStr),
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ===== Income Statement =====
// GET /reports/income-statement?from=2025-08-01&to=2025-08-31
func IncomeStatement(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Reports")
	if !ok {
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "from and to are required and must be in YYYY-MM-DD format",
				Code:        http.StatusBadRequest,
			},
			Message:   "from and to are required and must be in YYYY-MM-DD format",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	from, err := time.Parse(date, fromStr)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid from date for income statement report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidfrom,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	to, err := time.Parse(date, toStr)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid to date for income statement report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidto,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	revenue, expenses, tr, te, err := models.IncomeStatement(from, to)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate income statement report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	toLines := func(rows []dtos.IsRow) []dtos.IncomeStatementLine {
		out := make([]dtos.IncomeStatementLine, 0, len(rows))
		for _, r := range rows {
			out = append(out, dtos.IncomeStatementLine{
				AccountID:   r.AccountID,
				AccountCode: r.AccountCode,
				AccountName: r.AccountName,
				Amount:      r.Amount,
			})
		}
		return out
	}

	resp := dtos.IncomeStatementResponse{
		From:          from,
		To:            to,
		Revenue:       toLines(revenue),
		Expenses:      toLines(expenses),
		TotalRevenue:  tr,
		TotalExpenses: te,
		NetIncome:     tr - te,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: fmt.Sprintf("Income statement from %s to %s generated successfully", fromStr, toStr),
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Income statement",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ===== Cash Flow (Direct) =====
// POST /reports/cash-flow
// { "from":"2025-08-01", "to":"2025-08-31", "cash_account_ids":["<cash-account-id-1>", "..."] }
func CashFlow(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Reports")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.CashFlowRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Reports") {
		return
	}

	from, err := time.Parse(date, req.From)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid from date for cash flow report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidfrom,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	to, err := time.Parse(date, req.To)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid to date for cash flow report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidto,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	inflows, outflows, begin, end, err := models.CashFlow(from, to, req.CashAccountIDs)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate cash flow report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	resp := dtos.CashFlowResponse{
		From:          from,
		To:            to,
		TotalInflows:  inflows,
		TotalOutflows: outflows,
		NetChange:     inflows - outflows,
		BeginningCash: begin,
		EndingCash:    end,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Cash flow report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Cash flow",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// ===== General Ledger =====
// GET /reports/ledger/:account_id?from=YYYY-MM-DD&to=YYYY-MM-DD&page=1&size=50
func Ledger(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Reports")
	if !ok {
		return
	}
	accountID := mux.Vars(r)["account_id"]

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "from and to are required and in YYYY-MM-DD format",
				Code:        http.StatusBadRequest,
			},
			Message:   "from and to are required YYYY-MM-DD",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	from, err := time.Parse(date, fromStr)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid from date for ledger report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidfrom,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	to, err := time.Parse(date, toStr)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Invalid to date for ledger report",
				Code:        http.StatusBadRequest,
			},
			Message:   invalidto,
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	acct, opening, rows, total, err := models.Ledger(accountID, from, to, page, size)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to generate ledger report",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	// Compute running balance across returned page
	running := opening
	entries := make([]dtos.LedgerEntry, 0, len(rows))
	// Determine sign: debit-normal (Assets/Expenses) adds (debit-credit),
	// credit-normal (Liability/Equity/Income) adds (credit-debit)
	debitNormal := acct.Type == "Asset" || acct.Type == "Expense"

	for _, r := range rows {
		var delta float64
		if debitNormal {
			delta = r.Debit - r.Credit
		} else {
			delta = r.Credit - r.Debit
		}
		running += delta
		entries = append(entries, dtos.LedgerEntry{
			EntryID:        r.EntryID,
			EntryDate:      r.Date,
			Description:    r.Desc,
			Debit:          r.Debit,
			Credit:         r.Credit,
			RunningBalance: running,
		})
	}

	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	resp := dtos.LedgerResponse{
		AccountID:      acct.ID,
		AccountCode:    acct.Code,
		AccountName:    acct.Name,
		AccountType:    acct.Type,
		From:           from,
		To:             to,
		OpeningBalance: opening,
		Entries:        entries,
		ClosingBalance: running,
		Meta:           meta,
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Reports",
			Description: "Ledger report generated successfully",
			Code:        http.StatusOK,
		},
		Payload:   resp,
		Message:   "Ledger",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

func ExportAccountsCSVHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Reports")
	if !ok {
		return
	}
	accountType := r.URL.Query().Get("account_type")
	codePrefix := r.URL.Query().Get("code_prefix")

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=chart_of_accounts.csv")

	if err := models.ExportAccountsToCSV(w, accountType, codePrefix); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to export chart of accounts to CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
	}
}
func ExportJournalEntriesCSVHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "Reports")
	if !ok {
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	accountID := r.URL.Query().Get("account_id")

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=journal_entries.csv")

	if err := models.ExportJournalEntriesToCSV(w, startDate, endDate, accountID); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Reports",
				Description: "Failed to export journal entries to CSV",
				Code:        http.StatusInternalServerError,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
	}
}
