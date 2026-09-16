package models

import (
	"ekomasi_backend/dtos"
)

func UpdateVoucherPurchases(db DBExecutor, v dtos.BuyVoucherData, voucherID string) error {
	// Update purchase details (recipient info, message, delivery time, sender name)
	query := `
		UPDATE voucher_purchases SET  to_name = ?, to_email = ?, personalized_msg = ?, delivery_time = ?, from_name = ?
		WHERE voucher_id = ?
	`
	_, err := db.Exec(query, v.ToName, v.ToEmail, v.Message, v.DeliveryTime, v.FromName, voucherID)
	if err != nil {
		return err
	}
	return nil
}

// helper function to mark voucher status as aactive and update voucher order payment status to paid
// parameters:
//   - voucherID: string - The voucher ID to update
//
// returns:
//   - error: Database error or nil on success
func MarkVoucherAsPaid(db DBExecutor, voucherID string) error {
	// Update voucher status to active and payment status to paid
	query := `
		UPDATE vouchers SET status = 'active' WHERE voucher_id = ?
	`
	_, err := db.Exec(query, voucherID)
	if err != nil {
		return err
	}
	// update voucher order payment status to paid
	orderQuery := `
		UPDATE voucher_orders SET status = 'COMPLETED' WHERE voucher_id = ?
	`
	_, err = db.Exec(orderQuery, voucherID)
	if err != nil {
		return err
	}
	return nil
}
