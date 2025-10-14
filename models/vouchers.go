package models

import (
	"adenzo_backend/dtos"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"time"

	"github.com/teris-io/shortid"
)

func StringToTime(str string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range formats {
		t, err := time.Parse(layout, str)
		if err == nil {
			return t
		}
	}

	fmt.Println("Error parsing time:", str)
	return time.Time{}
}

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
	status := "inactive"
	if v.Status != nil {
		status = *v.Status
	}
	// generate unique code
	code, _ := GenerateVoucherCode()
	query := `
		INSERT INTO vouchers (voucher_id, code, original_value, status,user_id, expiry_date, balance)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := DB.Exec(query, voucherID, code, v.Amount, status, userID, v.ExpiryDate, v.Amount)
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := DB.Exec(query, purchaseID, voucherID, userID, v.ToName, v.ToEmail, v.Message, v.DeliveryTime, "PENDING", v.FromName)
	if err != nil {
		return err
	}
	return nil
}
func ListVouchers(page, size int, isRedeemed, status string) ([]dtos.VoucherData, *dtos.PaginationMeta, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	offset := (page - 1) * size

	// Build base query and args dynamically
	baseQuery := `
		FROM vouchers
		WHERE 1=1
	`
	args := []interface{}{}

	// Optional filters
	if isRedeemed != "" {
		baseQuery += " AND is_redeemed = ?"
		redeemed := isRedeemed == "true"
		args = append(args, redeemed)
	}

	if status != "" {
		baseQuery += " AND status = ?"
		args = append(args, status)
	}

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("failed to count vouchers: %w", err)
	}

	// Fetch vouchers with pagination
	selectQuery := `
		SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date, user_id, is_redeemed
	` + baseQuery + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, size, offset)

	rows, err := DB.Query(selectQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query vouchers: %w", err)
	}
	defer rows.Close()

	var vouchers []dtos.VoucherData
	for rows.Next() {
		var v dtos.VoucherData
		var userID string

		if err := rows.Scan(
			&v.VoucherID, &v.Code, &v.Balance, &v.Amount,
			&v.Status, &v.CreatedAt, &v.ExpiryDate, &userID, &v.IsReedemed,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan voucher: %w", err)
		}

		// Get participants
		v.To, v.From, err = getVoucherParticipants(v.VoucherID, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get voucher participants: %w", err)
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

func getVoucherParticipants(voucherID, userID string) (*string, *string, error) {
	log.Printf("getting voucher participants****")
	var phone, email, toEmail sql.NullString

	// Fetch sender details and recipient email
	err := DB.QueryRow(`
		SELECT u.phone_number, u.email, v.to_email
		FROM voucher_purchases v
		LEFT JOIN users u ON v.from_user_id = u.user_id
		WHERE v.voucher_id = ?
	`, voucherID).Scan(&phone, &email, &toEmail)

	if err != nil {
		if err == sql.ErrNoRows {
			// Fallback: get phone/email for provided userID if no voucher record found
			var fallbackPhone, fallbackEmail sql.NullString
			fallbackErr := DB.QueryRow(`
				SELECT phone_number, email FROM users WHERE user_id = ?
			`, userID).Scan(&fallbackPhone, &fallbackEmail)

			if fallbackErr != nil {
				return nil, nil, fallbackErr
			}

			if fallbackEmail.Valid {
				return &fallbackEmail.String, &fallbackEmail.String, nil
			}
			if fallbackPhone.Valid {
				return &fallbackPhone.String, &fallbackPhone.String, nil
			}
			return nil, nil, fmt.Errorf("no contact info found for user %s", userID)
		}
		return nil, nil, fmt.Errorf("failed to fetch voucher participants: %w", err)
	}

	// Prefer email if available; otherwise, use phone
	if email.Valid {
		return &email.String, &toEmail.String, nil
	}
	return &phone.String, &toEmail.String, nil
}

func GetVoucherByID(voucherID string) (dtos.SingleVoucherData, error) {
	err := isVoucherThere(voucherID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}
	var v dtos.SingleVoucherData
	var userID string
	query := `SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date, is_redeemed, user_id FROM vouchers WHERE voucher_id = ?`

	err = DB.QueryRow(query, voucherID).Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.Status, &v.CreatedAt, &v.ExpiryDate, &v.IsReedemed, &userID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}
	v.To, v.From, err = getVoucherParticipants(v.VoucherID, userID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}
	//get voucher history
	v.VoucherHistory, err = GetVoucherHistoryByVoucherID(voucherID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}
	return v, nil
}
func GetUserVoucherByID(voucherID, userID string) (dtos.SingleVoucherData, error) {
	err := isVoucherThere(voucherID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}
	var v dtos.SingleVoucherData
	query := `SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date, is_redeemed FROM vouchers WHERE voucher_id = ? AND user_id = ?`

	err = DB.QueryRow(query, voucherID, userID).Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.Status, &v.CreatedAt, &v.ExpiryDate, &v.IsReedemed)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}
	v.To, v.From, err = getVoucherParticipants(v.VoucherID, userID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
	}
	//get voucher history
	v.VoucherHistory, err = GetVoucherHistoryByVoucherID(voucherID)
	if err != nil {
		return dtos.SingleVoucherData{}, err
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
		SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date
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
		if err := rows.Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.Status, &v.CreatedAt, &v.ExpiryDate); err != nil {
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
	status := "active"
	if input.Status != nil {
		status = *input.Status
	}

	query := `
		UPDATE vouchers
		SET original_value = ?, expiry_date = ?, status = ?
		WHERE voucher_id = ?
	`
	_, err = DB.Exec(query, input.Amount, StringToTime(input.ExpiryDate), status, voucherID)
	if err != nil {
		return err
	}
	secondQuery := `
		UPDATE voucher_purchases
		SET to_name = ?, to_email = ?, personalized_msg = ?, delivery_time = ?, from_name = ?, notes = ?
		WHERE voucher_id = ?
	`
	_, err = DB.Exec(secondQuery, input.ToName, input.ToEmail, input.Message, StringToTime(input.ExpiryDate), input.FromName, input.InternalNotes, voucherID)

	if err != nil {
		return err
	}
	return nil
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
	if voucher.Status != "active" {
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
	query := `SELECT voucher_id, code, balance, original_value, status, created_at, expiry_date FROM vouchers WHERE code = ?`

	err := DB.QueryRow(query, code).Scan(&v.VoucherID, &v.Code, &v.Balance, &v.Amount, &v.Status, &v.CreatedAt, &v.ExpiryDate)
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

func CreateVoucherDesign(url string) (string, error) {
	designID, _ := shortid.Generate()
	// generate unique code
	query := `
		INSERT INTO voucher_designs (design_id, url)
		VALUES (?, ?)
	`
	_, err := DB.Exec(query, designID, url)
	if err != nil {
		return "", err
	}
	return designID, nil
}

func GetVoucherDesign(designID string) (string, error) {
	var url string
	query := `SELECT url FROM voucher_designs WHERE design_id = ? LIMIT 1`
	err := DB.QueryRow(query, designID).Scan(&url)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("design not found")
		}
		return "", err
	}
	return url, nil
}
func EditVoucherDesign(designID, newURL string) error {
	query := `UPDATE voucher_designs SET url = ? WHERE design_id = ?`
	res, err := DB.Exec(query, newURL, designID)
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no design found with ID %s", designID)
	}
	return nil
}
func DeleteVoucherDesign(designID string) error {
	query := `DELETE FROM voucher_designs WHERE design_id = ?`
	res, err := DB.Exec(query, designID)
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no design found with ID %s", designID)
	}
	return nil
}

type VoucherDesign struct {
	DesignID string `json:"design_id"`
	URL      string `json:"url"`
}

func GetAllVoucherDesigns() ([]VoucherDesign, error) {
	query := `SELECT design_id, url FROM voucher_designs`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var designs []VoucherDesign
	for rows.Next() {
		var d VoucherDesign
		if err := rows.Scan(&d.DesignID, &d.URL); err != nil {
			return nil, err
		}
		designs = append(designs, d)
	}
	return designs, nil
}

func GetVoucherHistoryByVoucherID(voucherID string) ([]map[string]any, error) {
	query := `
		SELECT history_id, redeemed_date, amount_redeemed, items_log
		FROM vouchers_history
		WHERE voucher_id = ?
		ORDER BY redeemed_date DESC
	`

	rows, err := DB.Query(query, voucherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []map[string]any

	for rows.Next() {
		var (
			historyID      string
			redeemedDate   time.Time
			amountRedeemed float64
			itemsLog       string
		)

		if err := rows.Scan(&historyID, &redeemedDate, &amountRedeemed, &itemsLog); err != nil {
			return nil, err
		}

		history := map[string]any{
			"history_id":      historyID,
			"redeemed_date":   redeemedDate,
			"amount_redeemed": amountRedeemed,
			"items_log":       itemsLog,
		}

		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return histories, nil
}
