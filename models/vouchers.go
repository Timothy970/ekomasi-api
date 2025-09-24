package models

import (
	"adenzo_backend/dtos"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"time"

	"github.com/teris-io/shortid"
)

func isVoucherThere(voucherID string) error {
	exists, err := RecordExists("vouchers", "voucher_id = ?", voucherID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("voucher not found")
	}
	return nil
}

func secureRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

// GenerateVoucherCode creates a secure voucher code like "AbCD-1a2b3c4d5e"
func GenerateVoucherCode() (string, error) {
	// generate 4 random letters
	prefix, err := secureRandomString(4)
	if err != nil {
		return "", err
	}

	// generate 6 random bytes (12 hex chars)
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil { // ✅ crypto/rand.Read
		return "", err
	}
	suffix := hex.EncodeToString(bytes)

	return fmt.Sprintf("%s-%s", prefix, suffix), nil
}
func CreateVoucherOrder(amount float64, voucherID string) (string, error) {
	voucherOrderID, _ := shortid.Generate()
	// generate unique code
	query := `
		INSERT INTO voucher_orders (voucher_order_id, voucher_id, amount, status, payment_method)
		VALUES (?, ?, ?, ?,?)
	`
	_, err := DB.Exec(query, voucherOrderID, voucherID, amount, "PENDING", "MPESA")
	if err != nil {
		return "", err
	}
	return voucherOrderID, nil
}
func AddNewVoucher(v dtos.Voucher, userID string) (string, error) {

	voucherID, _ := shortid.Generate()
	isActive := true
	if v.IsActive != nil {
		isActive = *v.IsActive
	}
	// generate unique code
	code, _ := GenerateVoucherCode()
	query := `
		INSERT INTO vouchers (voucher_id, code, original_value, is_active,user_id, expiry_date, balance)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := DB.Exec(query, voucherID, code, v.Amount, isActive, userID, v.ExpiryDate, v.Amount)
	if err != nil {
		return "", err
	}
	return voucherID, nil
}
func InsertIntoVoucherPurchases(v dtos.BuyVoucherData, userID, voucherID string) error {
	purchaseID, _ := shortid.Generate()
	// generate unique code
	query := `
		INSERT INTO voucher_purchases (purchase_id, voucher_id, from_user_id, to_name, to_email,personalized_msg, delivery_time, status, from_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := DB.Exec(query, purchaseID, voucherID, userID, v.ToName, v.ToEmail, v.Message, v.DeliveryTime, "PENDING", v.FromName)
	if err != nil {
		return err
	}
	return nil
}
func ListVouchers(page, size int) ([]dtos.VoucherData, *dtos.PaginationMeta, error) {
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
		SELECT voucher_id, code, balance, original_value, is_active, created_at, expiry_date
		FROM vouchers
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, size, offset)

	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var vouchers []dtos.VoucherData
	for rows.Next() {
		var v dtos.VoucherData
		err := rows.Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.IsActive, &v.CreatedAt, &v.ExpiryDate)
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
func GetVoucherByID(voucherID string) (dtos.VoucherData, error) {
	err := isVoucherThere(voucherID)
	if err != nil {
		return dtos.VoucherData{}, err
	}
	var v dtos.VoucherData
	query := `SELECT voucher_id, code, balance, original_value, is_active, created_at, expiry_date FROM vouchers WHERE voucher_id = ?`

	err = DB.QueryRow(query, voucherID).Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.IsActive, &v.CreatedAt, &v.ExpiryDate)
	if err != nil {
		return dtos.VoucherData{}, err
	}
	return v, nil
}
func GetUserVoucherByID(voucherID, userID string) (dtos.VoucherData, error) {
	err := isVoucherThere(voucherID)
	if err != nil {
		return dtos.VoucherData{}, err
	}
	var v dtos.VoucherData
	query := `SELECT voucher_id, code, balance, original_value, is_active, created_at, expiry_date FROM vouchers WHERE voucher_id = ? AND user_id = ?`

	err = DB.QueryRow(query, voucherID, userID).Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.IsActive, &v.CreatedAt, &v.ExpiryDate)
	if err != nil {
		return dtos.VoucherData{}, err
	}
	return v, nil
}
func GetUserVouchers(userID string, page, limit int) ([]dtos.VoucherData, *PaginationMeta, error) {

	offset := (page - 1) * limit

	// Count total vouchers
	var total int
	countQuery := `SELECT COUNT(*) FROM vouchers WHERE user_id = ?`
	if err := DB.QueryRow(countQuery, userID).Scan(&total); err != nil {
		return nil, nil, err
	}

	// Fetch vouchers with pagination
	query := `
		SELECT voucher_id, code, balance, original_value, is_active, created_at, expiry_date
		FROM vouchers
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := DB.Query(query, userID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var vouchers []dtos.VoucherData
	for rows.Next() {
		var v dtos.VoucherData
		if err := rows.Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.IsActive, &v.CreatedAt, &v.ExpiryDate); err != nil {
			return nil, nil, err
		}
		vouchers = append(vouchers, v)
	}

	// Calculate pagination metadata
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	pagination := &PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return vouchers, pagination, nil
}

func DeleteVoucher(voucherID string) error {
	err := isVoucherThere(voucherID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM vouchers WHERE voucher_id = ?`, voucherID)
	if err != nil {
		return err
	}
	return nil
}

func VoucherUpdate(input dtos.VoucherDataUpdate, voucherID string) error {
	err := isVoucherThere(voucherID)
	if err != nil {
		return err
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	balance := input.Amount
	if input.Balance != nil {
		balance = *input.Balance
	}
	query := `
		UPDATE vouchers
		SET original_value = ?, expiry_date = ?, is_active = ?, balance = ?
		WHERE voucher_id = ?
	`
	_, err = DB.Exec(query, input.Amount, input.ExpiryDate, isActive, balance, voucherID)
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
func isVoucherThereByCode(code string) error {
	exists, err := RecordExists("vouchers", "code = ?", code)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("voucher with this code not found")
	}
	return nil
}
func RedeemVoucher(code, userID string) (*dtos.VoucherData, error) {
	err := isVoucherThereByCode(code)
	if err != nil {
		return nil, err
	}
	voucher, err := GetVoucherByCode(code)
	if err != nil {
		return nil, err
	}
	if !voucher.IsActive {
		return nil, fmt.Errorf("voucher is not active")
	}
	if voucher.ExpiryDate.Before(time.Now()) {
		return nil, fmt.Errorf("voucher has expired")
	}
	//set user id to the one passed
	_, err = DB.Exec(`UPDATE vouchers SET user_id = ? WHERE code = ?`, userID, code)
	return &voucher, err
}

func GetVoucherByCode(code string) (dtos.VoucherData, error) {
	var v dtos.VoucherData
	query := `SELECT voucher_id, code, balance, original_value, is_active, created_at, expiry_date FROM vouchers WHERE code = ?`

	err := DB.QueryRow(query, code).Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.IsActive, &v.CreatedAt, &v.ExpiryDate)
	if err != nil {
		return dtos.VoucherData{}, err
	}
	return v, nil
}

// fetch voucher_id, from name, toname, to email, personalized msg, delivery time which is equal or less than now and status is PENDING from voucher_purchases table
// then get voucher original value from vouchers table using voucher_id
func GetUsersWithUnsentVoucherEmails() ([]dtos.VoucherEmailInfo, error) {
	rows, err := DB.Query(`
		SELECT vp.voucher_id, vp.from_name, vp.to_name, vp.to_email, vp.personalized_msg, vp.delivery_time, v.original_value, v.expiry_date, v.code
		FROM voucher_purchases vp
		JOIN vouchers v ON vp.voucher_id = v.voucher_id
		WHERE vp.status = 'PENDING' AND vp.delivery_time <= ?`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var infos []dtos.VoucherEmailInfo
	for rows.Next() {
		var info dtos.VoucherEmailInfo
		err := rows.Scan(&info.VoucherID, &info.FromName, &info.ToName, &info.ToEmail, &info.PersonalizedMsg, &info.DeliveryTime, &info.Amount, &info.ExpiryDate, &info.Code)
		if err != nil {
			return nil, err
		}
		log.Printf("message::::%s", info.PersonalizedMsg)
		infos = append(infos, info)
	}
	return infos, nil
}

// mark voucher email as sent by updating status to SENT in voucher_purchases table
func MarkVoucherEmailAsSent(voucherID string) error {
	_, err := DB.Exec(`UPDATE voucher_purchases SET status = 'SENT' WHERE voucher_id = ?`, voucherID)
	return err
}
