package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
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

func ProcessSplitPaymentHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.SplitPaymentRequest](r, w, requestSummary, start)
	if !ok {
		return
	}

	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "Payments") {
		return
	}

	if err := validateSplitPaymentMethods(req.PaymentMethods); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Invalid payment methods: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
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

	totalAmount := 0.0
	for _, paymentMethod := range req.PaymentMethods {
		totalAmount += paymentMethod.Amount
	}

	if totalAmount < order.TotalAmount {
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

	change := 0.0
	for _, paymentMethod := range req.PaymentMethods {
		switch strings.ToLower(paymentMethod.Type) {
		case "cash":
			if err := processCashPayment(order, &change, totalAmount, paymentMethod.Amount); err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Payments",
						Description: fmt.Sprintf("Failed to process cash payment: %s", err.Error()),
						Code:        http.StatusBadRequest,
					},
					Message:   "Failed to process cash payment",
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary,
				})
				return
			}

		case "mpesa":
			mpesaReq := &dtos.MpesaRequest{
				OrderID:     order.OrderID,
				Phone:       *paymentMethod.PhoneNumber,
				Amount:      int(paymentMethod.Amount),
				DeliveryID:  order.DeliveryID,
				Reference:   "ADENZO - " + order.OrderID,
				Description: fmt.Sprintf("Payment for order %s", order.OrderID),
			}

			client, err := NewMpesaClient()
			if err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Payments",
						Description: "Failed to initialize MPESA client",
						Code:        http.StatusInternalServerError,
					},
					Message:   fmt.Sprintf("Failed to initialize MPESA client: %s", err),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary,
				})
				return
			}

			response, err := client.LipaNaMpesaOnline(*mpesaReq)
			if err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Payments",
						Description: "Failed to initiate MPESA payment",
						Code:        http.StatusBadRequest,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary,
				})
				return
			}

			if err = models.StoreStkResponse(response, *mpesaReq); err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Payments",
						Description: "Failed to store MPESA payment request",
						Code:        http.StatusInternalServerError,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary,
				})
				return
			}

			if err = storeTransactionLog(*mpesaReq); err != nil {
				log.Printf("Failed to store transaction log: %v", err)
			}

		case "voucher":
			if err := processVoucherPayment(order, *paymentMethod.VoucherCode); err != nil {
				utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
					CollectiveInfo: utils.CollectiveInfo{
						Module:      "Payments",
						Description: fmt.Sprintf("Failed to process voucher payment: %s", err.Error()),
						Code:        http.StatusBadRequest,
					},
					Message:   err.Error(),
					TimeTaken: time.Since(start),
					Function:  utils.GetCurrentFuncName(),
					Request:   r,
					RawBody:   requestSummary,
				})
				return
			}

		default:
			utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
				CollectiveInfo: utils.CollectiveInfo{
					Module:      "Payments",
					Description: fmt.Sprintf("Unsupported payment method: %s", paymentMethod.Type),
					Code:        http.StatusBadRequest,
				},
				Message:   fmt.Sprintf("Unsupported payment method: %s", paymentMethod.Type),
				TimeTaken: time.Since(start),
				Function:  utils.GetCurrentFuncName(),
				Request:   r,
				RawBody:   requestSummary,
			})
			return
		}
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

func processCashPayment(order *dtos.Order, change *float64, totalAmount, paymentAmount float64) error {
	method := "CASH"
	paymentStatus := "SUCCESS"
	status := "SUCCESS"
	orderStatusData := dtos.UpdateOrderStatusRequest{
		PaymentMethod: &method,
		Status:        &status,
		PaymentStatus: &paymentStatus,
	}

	if err := models.UpdateOrderStatus(order.OrderID, orderStatusData); err != nil {
		return err
	}

	*change = totalAmount - order.TotalAmount
	if *change < 0 {
		*change = 0
	}

	logEntry := &dtos.TransactionsList{
		OrderID:              order.OrderID,
		TransactionReference: "ADENZO - " + order.OrderID,
		Amount:               order.TotalAmount,
		Status:               "COMPLETED",
		PaymentMethod:        "CASH",
	}

	if err := models.InsertTransaction(logEntry); err != nil {
		log.Printf("Failed to store transaction log: %v", err)
	}

	return nil
}

func processVoucherPayment(order *dtos.Order, voucherCode string) error {
	voucherBalance, err := models.ValidateVoucher(voucherCode)
	if err != nil {
		return fmt.Errorf("failed to validate voucher: %w", err)
	}

	if voucherBalance < order.TotalAmount {
		return fmt.Errorf("insufficient voucher balance")
	}

	method := "VOUCHER"
	paymentStatus := "SUCCESS"
	status := "SUCCESS"
	orderStatusData := dtos.UpdateOrderStatusRequest{
		PaymentMethod: &method,
		Status:        &status,
		PaymentStatus: &paymentStatus,
	}

	if err := models.UpdateOrderStatus(order.OrderID, orderStatusData); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if err := updateVoucherBalanceAndHistory(voucherCode, voucherBalance, order.TotalAmount, order); err != nil {
		return fmt.Errorf("failed to update voucher balance and history: %w", err)
	}

	logEntry := &dtos.TransactionsList{
		OrderID:              order.OrderID,
		TransactionReference: "ADENZO - " + order.OrderID,
		Amount:               order.TotalAmount,
		Status:               "COMPLETED",
		PaymentMethod:        "VOUCHER",
	}

	if err := models.InsertTransaction(logEntry); err != nil {
		log.Printf("Failed to store transaction log: %v", err)
	}

	return nil
}

func validateSplitPaymentMethods(methods []dtos.PaymentMethod) error {
	for _, method := range methods {
		switch strings.ToLower(method.Type) {
		case "mpesa", "airtel":
			if method.PhoneNumber == nil || *method.PhoneNumber == "" {
				return fmt.Errorf("phone number is required for mobile money payments")
			}

		case "voucher":
			if method.VoucherCode == nil || *method.VoucherCode == "" {
				return fmt.Errorf("voucher code is required for voucher payments")
			}
		}
	}
	return nil
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

func ProcessVoucherPaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Handler logic for processing cash payments
	start := time.Now()
	requestSummary := utils.GetRequestSummary(r)
	req, ok := DecodeRequestBody[dtos.VoucherPayment](r, w, requestSummary, start)
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

	voucherBalance, err := models.ValidateVoucher(req.VoucherCode)

	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Failed to validate voucher: %s", err.Error()),
				Code:        http.StatusBadRequest,
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}

	if voucherBalance < order.TotalAmount {
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
	paymentMethod := "VOUCHER"
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

	err = updateVoucherBalanceAndHistory(req.VoucherCode, voucherBalance, order.TotalAmount, order)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "Payments",
				Description: fmt.Sprintf("Failed to update voucher balance and history: %s", err.Error()),
				Code:        http.StatusInternalServerError,
			},
			Message:   "Failed to update voucher balance and history",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	//log the cash payment transaction
	logEntry := &dtos.TransactionsList{
		OrderID:              order.OrderID,
		TransactionReference: "ADENZO - " + order.OrderID,
		Amount:               order.TotalAmount,
		Status:               "COMPLETED",
		PaymentMethod:        "VOUCHER",
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
		},
		Message:   "Payment processed successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})

}

func updateVoucherBalanceAndHistory(voucherCode string, voucherBalance float64, orderTotalAmount float64, order *dtos.Order) error {
	cartItems := []dtos.CartItem{}
	cartItem := dtos.CartItem{}
	for _, item := range order.Items {
		product, _ := models.GetProductByID(item.ID)
		cartItem.Product = *product
		cartItem.Quantity = int(item.StockQuantity)
		cartItems = append(cartItems, cartItem)
	}

	if err := models.UpdateVoucherBalance(voucherCode, voucherBalance-orderTotalAmount); err != nil {
		return err
	}
	//add cart history
	if err := models.AddVoucherHistory(voucherCode, orderTotalAmount, cartItems); err != nil {
		return err
	}
	return nil
}
