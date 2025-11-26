package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func ScanProductsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)

	// Handler logic for scanning products
	barcode := r.URL.Query().Get("barcode")
	if barcode == "" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: "Failed to continue sacanning as barcode is empty",
				Code:        http.StatusBadRequest,
			},
			Message:   "Barcode is required and cannot be empty",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	product, err := models.GetProductThroughScanning(barcode)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: fmt.Sprintf("Failed to scan product: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to scan product",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Products",
			Description: "Product scanned successfully",
			Code:        http.StatusOK,
		},
		Payload:   product,
		Message:   "Product scanned successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func ProcessCashPaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Handler logic for processing cash payments
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.CashPayment](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Products") {
		return
	}
	order, err := models.GetOrderByID(req.OrderID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Failed to retrieve order: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to retrieve order",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	if req.Amount < order.TotalAmount {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: "Insufficient payment amount",
				Code:        http.StatusBadRequest,
			},
			Message:   "Insufficient payment amount",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	paymentMethod := "CASH"
	paymentStatus := "SUCCESS"
	status := "SUCCESS"
	orderStatusData := dtos.UpdateOrderStatusRequest{
		PaymentMethod: &paymentMethod,
		Status:        &status,
		PaymentStatus: &paymentStatus,
	}
	err = models.UpdateOrderStatus(order.OrderID, orderStatusData)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Failed to update order status: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to update order status",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	change := req.Amount - order.TotalAmount
	if change < 0 {
		change = 0
	}
	//log the cash payment transaction
	logEntry := &dtos.TransactionsList{
		OrderID:              order.OrderID,
		TransactionReference: "ADENZO - " + order.OrderID,
		Amount:               order.TotalAmount,
		Status:               "COMPLETED",
		PaymentMethod:        "CASH",
	}
	err = models.InsertTransaction(logEntry)
	if err != nil {
		log.Printf("Failed to store transaction log: %v", err)
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "Payments",
			Description: "Payment processed and order status updated successfully",
			Code:        http.StatusOK,
		},
		Payload: map[string]any{
			"order_id": order.OrderID,
			"change":   change,
		},
		Message:   "Payment processed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})

}

func PrintReceiptHandler(w http.ResponseWriter, r *http.Request) {
	// Handler logic for printing receipts
}

func DownloadReceiptHandler(w http.ResponseWriter, r *http.Request) {
	// Handler logic for downloading receipts
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	orderID := mux.Vars(r)["order_id"]
	order, err := models.GetOrderByID(orderID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Products",
				Description: fmt.Sprintf("Failed to retrieve order: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   "Failed to retrieve order",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	receiptData, err := models.GenerateReceiptPDF(*order)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Receipts",
				Description: fmt.Sprintf("Failed to generate receipt: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to generate receipt",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"receipt_%s.pdf\"", orderID))
	var buf bytes.Buffer
	err = receiptData.Output(&buf)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Receipts",
				Description: fmt.Sprintf("Failed to output PDF: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to output PDF",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	w.Write(buf.Bytes())
}
