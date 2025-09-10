package models

import (
	"adenzo_backend/dtos"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"github.com/teris-io/shortid"

	"time"
)

var nopayment = "payment not found"
var paymentid = "payment_id = ?"
var novoucher = "Voucher not found"

func StoreStkResponse(response map[string]interface{}, req dtos.MpesaRequest) error {
	// Extract values safely from the response map
	checkoutRequestID, _ := response["CheckoutRequestID"].(string)
	merchantRequestID, _ := response["MerchantRequestID"].(string)

	_, err := DB.Exec(`
		INSERT INTO stk_push_responses (
			order_id, delivery_id, amount, checkout_request_id,
			merchant_request_id, status
		)
		VALUES (?, ?, ?, ?, ?, 'PROCESSING')
	`, req.OrderID, req.DeliveryID, req.Amount, checkoutRequestID, merchantRequestID)

	if err != nil {
		return fmt.Errorf("failed to insert stk_push_response: %w", err)
	}

	return nil
}
func UpdateStkResponse(stk dtos.STKCallbackRequest, status string) (string, string, error) {
	// 1. Update status
	_, err := DB.Exec(`
		UPDATE stk_push_responses SET status = ?
		WHERE checkout_request_id = ? AND merchant_request_id = ?
	`, status, stk.Body.StkCallback.CheckoutRequestID, stk.Body.StkCallback.MerchantRequestID)
	if err != nil {
		return "", "", fmt.Errorf("failed to update status: %w", err)
	}

	// 2. Select delivery_id and order_id
	var deliveryID, orderID string
	err = DB.QueryRow(`
		SELECT delivery_id, order_id
		FROM stk_push_responses
		WHERE checkout_request_id = ? AND merchant_request_id = ? LIMIT 1
	`, stk.Body.StkCallback.CheckoutRequestID, stk.Body.StkCallback.MerchantRequestID).Scan(&deliveryID, &orderID)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch delivery/order ids: %w", err)
	}

	return deliveryID, orderID, nil
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
func GetVoucherByCode(code string) (*dtos.Voucher, error) {
	var (
		verificationHash string
		amount           float64
		isRedeemed       bool
		createdAt        time.Time
	)

	err := DB.QueryRow(`
		SELECT code, verification_hash, amount, is_redeemed, created_at
		FROM vouchers
		WHERE code = ?`, code).Scan(&code, &verificationHash, &amount, &isRedeemed, &createdAt)
	if err != nil {
		return nil, err
	}

	voucher := &dtos.Voucher{
		Code:             code,
		VerificationHash: verificationHash,
		Amount:           amount,
		IsRedeemed:       isRedeemed,
		CreatedAt:        createdAt,
	}

	return voucher, nil
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
func AddNewVoucher(v dtos.Voucher) error {
	exists, err := RecordExists("vouchers", "code = ?", v.Code)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("voucher with a similar code already exists")
	}
	hasher := sha256.New()
	hasher.Write([]byte(v.Code))
	hashSum := hasher.Sum(nil)

	voucherID, _ := shortid.Generate()

	query := `
		INSERT INTO vouchers (voucher_id, code, verification_hash, amount, is_redeemed)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(query, voucherID, v.Code, hex.EncodeToString(hashSum), v.Amount, false)
	if err != nil {
		return err
	}
	return nil
}
func ListVouchers(page, size int) ([]dtos.Voucher, *dtos.PaginationMeta, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	offset := (page - 1) * size

	var total int
	err := DB.QueryRow("SELECT COUNT(*) FROM vouchers").Scan(&total)
	if err != nil {
		return nil, nil, err
	}

	rows, err := DB.Query(`
		SELECT voucher_id, code, verification_hash, amount, is_redeemed, created_at, redeemed_at
		FROM vouchers
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, size, offset)

	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var vouchers []dtos.Voucher
	for rows.Next() {
		var v dtos.Voucher
		var redeemedAt sql.NullTime
		err := rows.Scan(&v.VoucherID, &v.Code, &v.VerificationHash, &v.Amount, &v.IsRedeemed, &v.CreatedAt, &redeemedAt)
		if redeemedAt.Valid {
			v.RedeemedAt = &redeemedAt.Time
		} else {
			v.RedeemedAt = nil
		}
		if err != nil {
			return nil, nil, err
		}
		vouchers = append(vouchers, v)
	}

	meta := dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: (total + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}
	return vouchers, &meta, nil

}
func GetVoucherByID(voucherID string) (dtos.Voucher, error) {
	var v dtos.Voucher
	query := `SELECT voucher_id, code, verification_hash, amount, is_redeemed, created_at, redeemed_at FROM vouchers WHERE voucher_id = ?`

	err := DB.QueryRow(query, voucherID).Scan(&v.VoucherID, &v.Code, &v.VerificationHash, &v.Amount, &v.IsRedeemed, &v.CreatedAt, &v.RedeemedAt)
	if err == sql.ErrNoRows {
		return dtos.Voucher{}, errors.New(novoucher)
	} else if err != nil {
		return dtos.Voucher{}, err
	}
	return v, nil
}
func DeleteVoucher(voucherID string) error {
	exists, err := RecordExists("vouchers", "voucher_id = ?", voucherID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(novoucher)
	}
	_, err = DB.Exec(`DELETE FROM vouchers WHERE voucher_id = ?`, voucherID)
	if err != nil {
		return err
	}
	return nil
}

func VoucherUpdate(input dtos.Voucher, voucherID string) error {
	exists, err := RecordExists("vouchers", "voucher_id = ?", voucherID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(novoucher)
	}
	query := `
		UPDATE vouchers
		SET code = ?, verification_hash = ?, amount = ?, is_redeemed = ?
		WHERE voucher_id = ?
	`
	_, err = DB.Exec(query, input.Code, input.VerificationHash, input.Amount, input.IsRedeemed, voucherID)
	return err
}
func isTransactionIDUnique(id string) error {
	exists, err := RecordExists("payments", "transaction_id = ?", id)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("duplicate transaction id")
	}
	return nil
}
