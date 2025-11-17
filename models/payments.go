package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/teris-io/shortid"
)

var nopayment = "payment not found"
var paymentid = "payment_id = ?"

func StoreStkResponse(response map[string]interface{}, req dtos.MpesaRequest) error {
	// Extract values safely from the response map
	checkoutRequestID, _ := response["CheckoutRequestID"].(string)
	merchantRequestID, _ := response["MerchantRequestID"].(string)
	var err error
	if strings.ToLower(req.Type) == "voucher" {
		_, err = DB.Exec(`
		INSERT INTO stk_push_responses (
			order_id, amount, checkout_request_id,
			merchant_request_id, status, type
		)
		VALUES (?, ?, ?, ?, 'PROCESSING', "VOUCHER")
	`, req.OrderID, req.Amount, checkoutRequestID, merchantRequestID)

		if err != nil {
			return err
		}
		return nil
	} else {
		_, err = DB.Exec(`
		INSERT INTO stk_push_responses (
			order_id, delivery_id, amount, checkout_request_id,
			merchant_request_id, status, type
		)
		VALUES (?, ?, ?, ?, ?, 'PROCESSING', "PRODUCT")
	`, req.OrderID, req.DeliveryID, req.Amount, checkoutRequestID, merchantRequestID)

		if err != nil {
			return err
		}

		return nil
	}
}
func UpdateStkResponse(stk dtos.STKCallbackRequest, status string) (string, string, string, error) {
	// 1. Update status
	_, err := DB.Exec(`
		UPDATE stk_push_responses SET status = ?
		WHERE checkout_request_id = ? AND merchant_request_id = ?
	`, status, stk.Body.StkCallback.CheckoutRequestID, stk.Body.StkCallback.MerchantRequestID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to update status: %w", err)
	}

	// 2. Select delivery_id and order_id
	var deliveryID, orderID, orderType sql.NullString
	err = DB.QueryRow(`
		SELECT delivery_id, order_id, type
		FROM stk_push_responses
		WHERE checkout_request_id = ? AND merchant_request_id = ? LIMIT 1
	`, stk.Body.StkCallback.CheckoutRequestID, stk.Body.StkCallback.MerchantRequestID).Scan(
		&deliveryID, &orderID, &orderType,
	)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to fetch delivery/order ids: %w", err)
	}

	// Convert NullString -> string ("" if NULL)
	return nullToString(deliveryID), nullToString(orderID), nullToString(orderType), nil
}

func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
func UpdateDeliveryOrderTables(deliveryID, orderId string) error {
	// Update order status
	_, err := DB.Exec(`
		UPDATE orders SET status = 'PAID'
		WHERE order_id = ?
	`, orderId)
	if err != nil {
		return fmt.Errorf("failed to update orders table: %w", err)
	}

	// Update delivery status
	_, err = DB.Exec(`
		UPDATE deliveries SET status = 'PAID'
		WHERE delivery_id = ?
	`, deliveryID)
	if err != nil {
		return fmt.Errorf("failed to update deliveries table: %w", err)
	}

	return nil
}

func UpdateVoucherOrderTables(orderId string) error {
	// Update order status
	newStatus := "COMPLETED"

	_, err := DB.Exec(`
    UPDATE voucher_orders SET status = ?
    WHERE voucher_order_id = ?
`, newStatus, orderId)
	if err != nil {
		return fmt.Errorf("failed to update voucher order status: %w", err)
	}
	//get vourcher id
	var voucherID string
	err = DB.QueryRow(`
	SELECT voucher_id FROM voucher_orders WHERE voucher_order_id = ?
`, orderId).Scan(&voucherID)
	// Update voucher status
	isActive := true
	_, err = DB.Exec(`
		UPDATE vouchers SET is_active = ?
		WHERE voucher_id = ?
	`, isActive, voucherID)
	if err != nil {
		return fmt.Errorf("failed to update deliveries table: %w", err)
	}

	return nil
}
func SetVoucherAsRedeemed(code string) error {
	_, err := DB.Exec(`
	UPDATE vouchers SET is_redeemed = true WHERE code = ?`, code)
	if err != nil {
		return err
	}
	return nil
}

func CreatePayment(p dtos.Payment) error {
	err := isOrderThere(p.OrderID)
	if err != nil {
		return err
	}
	err = isTransactionIDUnique(p.TransactionID)
	if err != nil {
		return err
	}
	paymentID, _ := shortid.Generate()
	status := "PENDING"
	if p.Status != "" {
		status = p.Status
	}
	query := `
		INSERT INTO payments (payment_id, order_id, amount, voucher_id, status, payment_method, transaction_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, paymentID, p.OrderID, p.Amount, p.VoucherID, status, p.PaymentMethod, p.TransactionID)
	return err
}
func isPaymentThere(id string) error {
	exists, err := RecordExists("payments", paymentid, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(nopayment)
	}
	return nil
}
func GetPaymentByID(paymentID string) (*dtos.Payment, error) {
	err := isPaymentThere(paymentID)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT payment_id, order_id, amount, voucher_id, status, payment_method, transaction_id, created_at
		FROM payments
		WHERE payment_id = ?`

	var p dtos.Payment
	err = DB.QueryRow(query, paymentID).Scan(
		&p.PaymentID, &p.OrderID, &p.Amount, &p.VoucherID,
		&p.Status, &p.PaymentMethod, &p.TransactionID, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func ListPayments(page, limit int) ([]dtos.Payment, *dtos.PaginationMeta, error) {
	offset := (page - 1) * limit

	var totalItems int
	err := DB.QueryRow("SELECT COUNT(*) FROM payments").Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	query := `
		SELECT payment_id, order_id, amount, voucher_id, status, payment_method, transaction_id, created_at
		FROM payments
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := DB.Query(query, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var payments []dtos.Payment
	for rows.Next() {
		var p dtos.Payment
		err := rows.Scan(&p.PaymentID, &p.OrderID, &p.Amount, &p.VoucherID, &p.Status, &p.PaymentMethod, &p.TransactionID, &p.CreatedAt)
		if err != nil {
			return nil, nil, err
		}
		payments = append(payments, p)
	}

	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: int(math.Ceil(float64(totalItems) / float64(limit))),
		HasPrev:    page > 1,
		HasNext:    page*limit < totalItems,
	}

	return payments, meta, nil
}

func UpdatePayment(p dtos.PaymentUpdate, paymentID string) error {
	err := isPaymentThere(paymentID)
	if err != nil {
		return err
	}
	query := `
		UPDATE payments
		SET status = ?
		WHERE payment_id = ?`

	_, err = DB.Exec(query, p.Status, paymentID)
	return err
}

func DeletePayment(paymentID string) error {
	err := isPaymentThere(paymentID)
	if err != nil {
		return err
	}
	query := `DELETE FROM payments WHERE payment_id = ?`
	_, err = DB.Exec(query, paymentID)
	return err
}

func AddRefundRequest(refund dtos.Refund, userID string) error {
	//check if order exists
	exists, err := RecordExists("orders", "order_id = ?", refund.OrderID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("order not found")
	}
	query := `INSERT INTO refunds (user_id, order_id, amount, reason, status)
				  VALUES (?, ?, ?, ?, ?)
	`

	_, err = DB.Exec(query, userID, refund.OrderID, refund.Amount, refund.Reason, refund.Status)
	if err != nil {
		return err
	}

	return nil
}

func ProcessRefund(req dtos.RefundPayload, refundID string) error {
	//check if refund exists
	exists, err := RecordExists("refunds", whereID, refundID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("refund not found")
	}
	query := `UPDATE refunds SET status = ? WHERE id = ?`
	_, err = DB.Exec(query, req.Status, refundID)
	return err
}

func ListRefunds(page, size int) ([]dtos.Refund, *dtos.PaginationMeta, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	offset := (page - 1) * size

	// Count total refunds
	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	// Fetch paginated refunds
	rows, err := DB.Query(`
		SELECT id, user_id, order_id, amount, status, created_at
		FROM refunds
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var refunds []dtos.Refund
	for rows.Next() {
		var refund dtos.Refund
		if err := rows.Scan(&refund.ID, &refund.UserID, &refund.OrderID, &refund.Amount, &refund.Status, &refund.CreatedAt); err != nil {
			return nil, nil, err
		}
		refunds = append(refunds, refund)
	}

	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return refunds, meta, nil
}

func GetRefundByID(id int) (*dtos.Refund, error) {
	var refund dtos.Refund
	err := DB.QueryRow(`
		SELECT id, user_id, order_id, amount, status, created_at
		FROM refunds
		WHERE user_id = ?`, id).Scan(
		&refund.ID, &refund.UserID, &refund.OrderID, &refund.Amount, &refund.Status, &refund.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("refund not found")
	} else if err != nil {
		return nil, err
	}
	return &refund, nil
}
func AddVoucherHistory(code string, amount float64, itemsLog []dtos.OrderProduct) error {
	historyID, _ := shortid.Generate()
	jsonData, err := json.Marshal(itemsLog)
	if err != nil {
		return err
	}
	err = IsVoucherThereByCode(code)
	if err != nil {
		return err
	}
	voucher, err := GetVoucherByCode(code)
	query := `
		INSERT INTO vouchers_history (history_id, voucher_id, amount_redeemed, items_log)
		VALUES (?, ?, ?, ?)
	`

	_, err = DB.Exec(query, historyID, voucher.VoucherID, amount, jsonData)
	if err != nil {
		return fmt.Errorf("failed to insert voucher history: %v", err)
	}

	return nil
}

func CreateMpesaPaybill(paybill dtos.MpesaPaybill) error {
	paybillID, _ := shortid.Generate()
	query := `
		INSERT INTO mpesa_paybills (id, paybill_number, account_reference)
		VALUES (?, ?, ?)`
	_, err := DB.Exec(query, paybillID, paybill.PaybillNumber, paybill.AccountReference)
	return err
}

func ListMpesaPaybills(searchParam string, page, size int) ([]dtos.MpesaPaybill, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	countQuery := `SELECT COUNT(*) FROM mpesa_paybills`
	var countArgs []interface{}

	if searchParam != "" {
		countQuery += " WHERE paybill_number LIKE ?"
		countArgs = append(countArgs, "%"+searchParam+"%")
	}

	var totalItems int
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}
	query := `
		SELECT id, paybill_number, account_reference, status, created_at
		FROM mpesa_paybills
	`
	var queryArgs []interface{}

	if searchParam != "" {
		query += " WHERE paybill_number LIKE ?"
		queryArgs = append(queryArgs, "%"+searchParam+"%")
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs = append(queryArgs, size, offset)

	rows, err := DB.Query(query, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var paybills []dtos.MpesaPaybill
	for rows.Next() {
		var p dtos.MpesaPaybill
		if err := rows.Scan(&p.ID, &p.PaybillNumber, &p.AccountReference, &p.Status, &p.CreatedAt); err != nil {
			return nil, nil, err
		}
		paybills = append(paybills, p)
	}
	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: totalItems,
		TotalPages: int(math.Ceil(float64(totalItems) / float64(size))),
		HasPrev:    page > 1,
		HasNext:    page*size < totalItems,
	}

	return paybills, &meta, nil
}

func isPayBillThere(id string) error {
	exists, err := RecordExists("mpesa_paybills", "id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("mpesa paybill not found")
	}
	return nil
}
func GetMpesaPaybillByID(id string) (*dtos.MpesaPaybill, error) {
	err := isPayBillThere(id)
	if err != nil {
		return nil, err
	}
	var p dtos.MpesaPaybill
	err = DB.QueryRow(`SELECT id, paybill_number, account_reference, status, created_at
		FROM mpesa_paybills WHERE id = ?`, id).Scan(&p.ID, &p.PaybillNumber, &p.AccountReference, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func UpdateMpesaPaybill(id string, paybill dtos.MpesaPaybillUpdate) error {
	err := isPayBillThere(id)
	if err != nil {
		return err
	}
	query := `
		UPDATE mpesa_paybills
		SET paybill_number = ?, account_reference = ?, status = ?
		WHERE id = ?`
	_, err = DB.Exec(query, paybill.PaybillNumber, paybill.AccountReference, paybill.Status, id)
	return err
}

func DeleteMpesaPaybill(id string) error {
	err := isPayBillThere(id)
	if err != nil {
		return err
	}
	query := `DELETE FROM mpesa_paybills WHERE id = ?`
	_, err = DB.Exec(query, id)
	return err
}
