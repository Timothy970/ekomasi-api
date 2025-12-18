package models

import (
	"adenzo_backend/dtos"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
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
func StringToBool(str string) bool {
	return str == "true" || str == "1"
}
func StringToFloat64(str string) float64 {
	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}
	return value
}
func IsVoucherThere(voucherID string) error {
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
func CreateVoucherOrder(amount float64, voucherID string, paymentMethod string) (string, error) {
	voucherOrderID, _ := shortid.Generate()
	// generate unique code
	query := `
		INSERT INTO voucher_orders (voucher_order_id, voucher_id, amount, status, payment_method)
		VALUES (?, ?, ?, ?,?)
	`
	_, err := DB.Exec(query, voucherOrderID, voucherID, amount, "PENDING", strings.ToUpper(paymentMethod))
	if err != nil {
		return "", err
	}
	return voucherOrderID, nil
}
func UpdateVoucherPurchaseAmount(amount float64, voucherID string) (string, error) {
	query := `
		UPDATE voucher_orders SET amount = amount + ? WHERE voucher_id = ?
	`
	_, err := DB.Exec(query, amount, voucherID)
	if err != nil {
		return "", err
	}
	selectQuery := `
		SELECT voucher_order_id FROM voucher_orders WHERE voucher_id = ?
	`
	var voucherOrderID string
	err = DB.QueryRow(selectQuery, voucherID).Scan(&voucherOrderID)
	if err != nil {
		return "", err
	}
	return voucherOrderID, nil
}
func AddNewVoucher(v dtos.Voucher, userID string) (string, error) {
	err := ValidateDesignID(*v.DesignID)
	if err != nil {
		return "", err
	}
	voucherID, _ := shortid.Generate()
	status := "inactive"
	if v.Status != nil {
		status = *v.Status
	}
	// generate unique code
	code, _ := GenerateVoucherCode()
	query := `
		INSERT INTO vouchers (voucher_id, code, original_value, status,user_id, expiry_date, balance, design_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(query, voucherID, code, v.Amount, status, userID, v.ExpiryDate, v.Amount, v.DesignID)
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
func ListVouchers(page, size int, isRedeemed, status, code, customer string) ([]dtos.VoucherData, *dtos.PaginationMeta, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	offset := (page - 1) * size

	// Build base query and args dynamically
	baseQuery := `
		FROM vouchers v
		LEFT JOIN voucher_purchases vp ON v.voucher_id = vp.voucher_id
		WHERE 1=1
	`
	args := []interface{}{}

	// Optional filters
	if isRedeemed != "" {
		baseQuery += " AND v.is_redeemed = ?"
		redeemed := isRedeemed == "true"
		args = append(args, redeemed)
	}

	if status != "" {
		baseQuery += " AND v.status = ?"
		args = append(args, status)
	}
	if code != "" {
		baseQuery += " AND v.code LIKE ?"
		args = append(args, "%"+code+"%")
	}
	if customer != "" {
		baseQuery += " AND (vp.to_email LIKE ? OR vp.to_name LIKE ? OR vp.from_name LIKE ?)"
		args = append(args, "%"+customer+"%", "%"+customer+"%", "%"+customer+"%")
	}

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("failed to count vouchers: %w", err)
	}

	// Fetch vouchers with pagination
	selectQuery := `
		SELECT v.voucher_id, v.code, v.balance, v.original_value, v.status, v.created_at, v.expiry_date, v.user_id, v.is_redeemed
	` + baseQuery + `
		ORDER BY v.created_at DESC
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
	err := IsVoucherThere(voucherID)
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
	err := IsVoucherThere(voucherID)
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
	err := IsVoucherThere(voucherID)
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
	err := IsVoucherThere(voucherID)
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
func IsVoucherThereByCode(code string) error {
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
	err := IsVoucherThereByCode(code)
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

func CreateVoucherDesign(url, name, status string) (string, error) {
	designID, _ := shortid.Generate()
	// generate unique code
	query := `
		INSERT INTO voucher_designs (design_id, url, name, status)
		VALUES (?, ?, ?, ?)
	`
	_, err := DB.Exec(query, designID, url, name, status)
	if err != nil {
		return "", err
	}
	return designID, nil
}

func GetVoucherDesign(designID string) (dtos.VoucherDesign, error) {
	err := isVoucherDesignThere(designID)
	if err != nil {
		return dtos.VoucherDesign{}, err
	}
	var design dtos.VoucherDesign
	query := `SELECT url, name, status, created_at FROM voucher_designs WHERE design_id = ? LIMIT 1`
	err = DB.QueryRow(query, designID).Scan(&design.URL, &design.Name, &design.Status, &design.Created_At)
	if err != nil {
		if err == sql.ErrNoRows {
			return dtos.VoucherDesign{}, fmt.Errorf("design not found")
		}
		return dtos.VoucherDesign{}, err
	}
	return design, nil
}
func EditVoucherDesign(designID string, newURL *string, newName, newStatus string) error {
	err := isVoucherDesignThere(designID)
	if err != nil {
		return err
	}
	
	query := `UPDATE voucher_designs SET name = ?, status = ?`
	args := []interface{}{newName, newStatus}
	
	if newURL != nil && *newURL != "" {
		query += `, url = ?`
		args = append(args, *newURL)
	}
	
	query += ` WHERE design_id = ?`
	args = append(args, designID)
	
	_, err = DB.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}

func DeleteVoucherDesign(designID string) error {
	err := isVoucherDesignThere(designID)
	if err != nil {
		return err
	}
	query := `DELETE FROM voucher_designs WHERE design_id = ?`
	_, err = DB.Exec(query, designID)
	if err != nil {
		return err
	}

	return nil
}

func GetAllVoucherDesigns(page, size int, name, status string) ([]dtos.VoucherDesign, *dtos.PaginationMeta, error) {
	var (
		total int
		args  []interface{}
	)

	countQuery := `SELECT COUNT(*) FROM voucher_designs`
	if name != "" {
		countQuery += " WHERE name LIKE ?"
		args = append(args, "%"+name+"%")
	}
	if status != "" {
		if len(args) == 0 {
			countQuery += " WHERE status = ?"
		} else {
			countQuery += " AND status = ?"
		}
		args = append(args, status)
	}

	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count query failed: %w", err)
	}

	selectQuery := `
		SELECT design_id, url, created_at, name, status
		FROM voucher_designs
	`
	var queryArgs []interface{}

	if name != "" {
		selectQuery += " WHERE name LIKE ?"
		queryArgs = append(queryArgs, "%"+name+"%")
	}
	if status != "" {
		if len(queryArgs) == 0 {
			selectQuery += " WHERE status = ?"
		} else {
			selectQuery += " AND status = ?"
		}
		queryArgs = append(queryArgs, status)
	}

	selectQuery += " ORDER BY created_at DESC LIMIT ?, ?"

	// Add pagination
	queryArgs = append(queryArgs, (page-1)*size, size)

	rows, err := DB.Query(selectQuery, queryArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("select query failed: %w", err)
	}
	defer rows.Close()

	// ----- SCAN RESULTS -----
	var designs []dtos.VoucherDesign
	for rows.Next() {
		var d dtos.VoucherDesign
		if err := rows.Scan(
			&d.DesignID,
			&d.URL,
			&d.Created_At,
			&d.Name,
			&d.Status,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}
		designs = append(designs, d)
	}

	// ----- PAGINATION META -----
	meta := &dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: (total + size - 1) / size,
		HasPrev:    page > 1,
		HasNext:    page*size < total,
	}

	return designs, meta, nil
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
		//convert itemsLog from json string to map[string]any
		var itemsLogSlice []map[string]any
		if itemsLog != "" {
			itemsLogSlice = make([]map[string]any, 0)
			if err := json.Unmarshal([]byte(itemsLog), &itemsLogSlice); err != nil {
				return nil, err
			}
		}

		history := map[string]any{
			"history_id":      historyID,
			"redeemed_date":   redeemedDate,
			"amount_redeemed": amountRedeemed,
			"items_log":       itemsLogSlice,
		}

		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return histories, nil
}
func isVoucherDesignThere(designID string) error {
	exists, err := RecordExists("voucher_designs", "design_id = ?", designID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("voucher design not found")
	}
	//check if design is active

	return nil
}
func IsVoucherDesignActive(designID string) error {
	var status string
	err := DB.QueryRow(`SELECT status FROM voucher_designs WHERE design_id = ?`, designID).Scan(&status)
	if err != nil {
		return err
	}
	if status != "active" {
		return errors.New("voucher design is not active")
	}
	return nil
}

func ValidateDesignID(designID string) error {
	err := isVoucherDesignThere(designID)
	if err != nil {
		return err
	}
	err = IsVoucherDesignActive(designID)
	if err != nil {
		return err
	}
	return nil
}

func CreateNewVoucher(v dtos.VoucherDataCreate, userID string) (string, error) {
	err := isVoucherDesignThere(v.DesignID)
	if err != nil {
		return "", err
	}
	// Generate a new voucher ID
	voucherID, _ := shortid.Generate()
	code, _ := GenerateVoucherCode()
	// Insert the new voucher into the database
	query := `
		INSERT INTO vouchers (voucher_id, design_id, user_id, code, balance, original_value, expiry_date, status, is_redeemed, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(query, voucherID, v.DesignID, userID, code, v.Amount, v.Amount, StringToTime(v.ExpiryDate), "active", false, v.InternalNotes)
	if err != nil {
		return "", err
	}
	deliveryTime := StringToTime(v.DeliveryTime)
	//insert into voucher_purchases table
	err = InsertIntoVoucherPurchases(dtos.BuyVoucherData{
		DesignID:     v.DesignID,
		Amount:       v.Amount,
		FromName:     v.FromName,
		ToName:       v.ToName,
		ToEmail:      v.ToEmail,
		Message:      v.Message,
		DeliveryTime: deliveryTime.Format("2006-01-02 15:04:05"),
	}, userID, voucherID)
	return voucherID, nil
}

func ListVoucherPurchases(page, size int, name string) ([]dtos.VoucherPurchaseData, *dtos.PaginationMeta, error) {
	offset := (page - 1) * size

	baseQuery := `
		FROM voucher_purchases vp
		JOIN vouchers v ON vp.voucher_id = v.voucher_id
		LEFT JOIN voucher_designs vd ON v.design_id = vd.design_id
		WHERE 1=1
	`
	args := []interface{}{}

	if name != "" {
		baseQuery += " AND (vp.from_name LIKE ? OR vp.to_name LIKE ? OR vp.to_email LIKE ?)"
		nameLike := "%" + name + "%"
		args = append(args, nameLike, nameLike, nameLike)
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("failed to count voucher purchases: %w", err)
	}

	var voucherUpdateTime time.Time
	selectQuery := `
		SELECT 
			v.voucher_id, v.code, v.balance, v.original_value, 
			vp.from_name, vp.to_name, vp.to_email, vp.personalized_msg, vp.from_user_id, vd.url,
			vp.created_at, v.updated_at
	` + baseQuery + `
		ORDER BY v.updated_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, size, offset)

	rows, err := DB.Query(selectQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query voucher purchases: %w", err)
	}
	defer rows.Close()

	var vouchers []dtos.VoucherPurchaseData
	for rows.Next() {
		var v dtos.VoucherPurchaseData
		var personalizedMsg sql.NullString
		var designURL sql.NullString
		var createdAt time.Time
		var fromUserID string
		if err := rows.Scan(
			&v.VoucherID,
			&v.Code,
			&v.Balance,
			&v.Amount,
			&v.FromName,
			&v.ToName,
			&v.ToEmail,
			&personalizedMsg,
			&fromUserID,
			&designURL,
			&createdAt,
			&voucherUpdateTime,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan voucher purchase row: %w", err)
		}
		if personalizedMsg.Valid {
			personalizedMsgStr := personalizedMsg.String
			v.Message = &personalizedMsgStr
		}
		if designURL.Valid {
			designURLStr := designURL.String
			v.DesignURL = &designURLStr
		}
		v.CreatedAt = &createdAt

		v.FromEmail, _, err = getVoucherParticipants(v.VoucherID, fromUserID)
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
func GetVoucherPurchases(voucherID string) (dtos.VoucherPurchaseData, error) {
	var (
		result          dtos.VoucherPurchaseData
		personalizedMsg sql.NullString
		designURL       sql.NullString
		fromUserID      string
		createdAt       time.Time
		deliveryTime    time.Time
	)

	baseQuery := `
		FROM voucher_purchases vp
		JOIN vouchers v ON vp.voucher_id = v.voucher_id
		LEFT JOIN voucher_designs vd ON v.design_id = vd.design_id
		WHERE v.voucher_id = ?
	`

	selectQuery := `
		SELECT 
			v.voucher_id,
			v.code,
			v.balance,
			v.original_value,
			vp.from_name,
			vp.to_name,
			vp.to_email,
			vp.personalized_msg,
			vp.from_user_id,
			vd.url,
			vp.created_at,
			vp.delivery_time
	` + baseQuery + `
		ORDER BY vp.created_at DESC
	`

	err := DB.QueryRow(selectQuery, voucherID).Scan(
		&result.VoucherID,
		&result.Code,
		&result.Balance,
		&result.Amount,
		&result.FromName,
		&result.ToName,
		&result.ToEmail,
		&personalizedMsg,
		&fromUserID,
		&designURL,
		&createdAt,
		&deliveryTime,
	)

	if err != nil {
		return dtos.VoucherPurchaseData{}, fmt.Errorf("failed to scan voucher purchase row: %w", err)
	}

	// Handle nullable fields
	if personalizedMsg.Valid {
		msg := personalizedMsg.String
		result.Message = &msg
	}

	if designURL.Valid {
		url := designURL.String
		result.DesignURL = &url
	}

	result.CreatedAt = &createdAt
	if !deliveryTime.IsZero() {
		deliveryTimeStr := deliveryTime.Format("2006-01-02")
		result.DeliveryTime = &deliveryTimeStr
	}

	// Fetch participants (e.g. sender email)
	result.FromEmail, _, err = getVoucherParticipants(result.VoucherID, fromUserID)
	if err != nil {
		return dtos.VoucherPurchaseData{}, fmt.Errorf("failed to get voucher participants: %w", err)
	}

	return result, nil
}

func UpdateVoucher(voucherID string, amount float64, status string, designID string) error {
	err := IsVoucherThere(voucherID)
	if err != nil {
		return err
	}
	query := `
		UPDATE vouchers SET balance = balance + ?, status = ?, design_id = ?
		WHERE voucher_id = ?
	`
	_, err = DB.Exec(query, amount, status, designID, voucherID)
	if err != nil {
		return err
	}
	return nil
}

func UpdateVoucherPurchases(v dtos.BuyVoucherData, voucherID string) error {
	query := `
		UPDATE voucher_purchases SET  to_name = ?, to_email = ?, personalized_msg = ?, delivery_time = ?, from_name = ?
		WHERE voucher_id = ?
	`
	_, err := DB.Exec(query, v.ToName, v.ToEmail, v.Message, v.DeliveryTime, v.FromName, voucherID)
	if err != nil {
		return err
	}
	return nil
}
