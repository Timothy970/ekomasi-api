package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

func isReviewThere(reviewID string) error {
	exists, err := RecordExists("product_reviews", "review_id = ?", reviewID)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("review not found")
	}
	return nil
}
func GetProductReview(productID, reviewID string, limit, page int) ([]dtos.ReviewResponse, *dtos.PaginationMeta, error) {
	err := IsProductThere(productID)
	if err != nil {
		return nil, nil, err
	}
	err = isReviewThere(reviewID)

	if err != nil {
		return nil, nil, err
	}
	query := `
			SELECT review_id, user_id, score, details, created_at
			FROM product_reviews
			WHERE product_id = ? AND review_id = ?
			LIMIT 1
		`

	var r dtos.ReviewResponse
	var userID string
	err = DB.QueryRow(query, productID, reviewID).
		Scan(&r.ID, &userID, &r.Score, &r.Details, &r.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("review for the product not found")
		}
		return nil, nil, err
	}
	r.User, err = GetUserDisplayName(userID)
	if err != nil {
		return nil, nil, err
	}
	return []dtos.ReviewResponse{r}, nil, nil
}

func GetProductReviews(productID, sortBy string, rating, limit, page int) (*dtos.DetailedReviewResponse, *dtos.PaginationMeta, error) {

	// Pagination
	offset := (page - 1) * limit

	// ----- COUNT TOTAL ITEMS -----
	countArgs := []interface{}{productID}
	countQuery := `
		SELECT COUNT(*)
		FROM product_reviews
		WHERE product_id = ?
	`
	if rating > 0 {
		countQuery += " AND score = ?"
		countArgs = append(countArgs, rating)
	}

	var totalItems int
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&totalItems); err != nil {
		return nil, nil, err
	}

	// ----- FETCH PAGINATED REVIEWS -----
	queryArgs := []interface{}{productID}
	query := `
		SELECT review_id, user_id, score, details, created_at
		FROM product_reviews
		WHERE product_id = ?
	`

	if rating > 0 {
		query += " AND score = ?"
		queryArgs = append(queryArgs, rating)
	}

	// Sorting
	switch strings.ToLower(sortBy) {
	case "newest":
		query += " ORDER BY created_at DESC"
	case "oldest":
		query += " ORDER BY created_at ASC"
	case "highest":
		query += " ORDER BY score DESC"
	case "lowest":
		query += " ORDER BY score ASC"
	default:
		query += " ORDER BY created_at DESC"
	}

	query += " LIMIT ? OFFSET ?"
	queryArgs = append(queryArgs, limit, offset)

	rows, err := DB.Query(query, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// --- REVIEWS ARRAY ---
	var reviewList []dtos.ReviewResponse

	for rows.Next() {
		var r dtos.ReviewResponse
		var userID string
		if err := rows.Scan(&r.ID, &userID, &r.Score, &r.Details, &r.CreatedAt); err != nil {
			return nil, nil, err
		}
		r.User, err = GetUserDisplayName(userID)
		if err != nil {
			return nil, nil, err
		}
		reviewList = append(reviewList, r)
	}

	// ----- SCORE COUNTS (1–5 ALWAYS RETURNED) -----
	scoreQuery := `
		SELECT score, COUNT(*)
		FROM product_reviews
		WHERE product_id = ?
		GROUP BY score
	`

	scoreRows, err := DB.Query(scoreQuery, productID)
	if err != nil {
		return nil, nil, err
	}
	defer scoreRows.Close()

	scoreMap := make(map[int]int)
	totalScore := 0

	for scoreRows.Next() {
		var score, count int
		if err := scoreRows.Scan(&score, &count); err != nil {
			return nil, nil, err
		}
		scoreMap[score] = count
		totalScore += score * count
	}

	// Build score counts 1–5 (missing = 0)
	finalScoreCounts := make([]dtos.ScoreCount, 0, 5)
	for i := 1; i <= 5; i++ {
		finalScoreCounts = append(finalScoreCounts, dtos.ScoreCount{
			Score: i,
			Count: scoreMap[i],
		})
	}

	// Average score
	averageScore := 0.0
	if totalItems > 0 {
		averageScore = float64(totalScore) / float64(totalItems)
	}

	// ----- BUILD DETAILED RESPONSE -----
	detailed := &dtos.DetailedReviewResponse{
		Reviews:      reviewList,
		AverageScore: averageScore,
		ScoreCounts:  finalScoreCounts,
	}

	// ----- Pagination -----
	totalPages := (totalItems + limit - 1) / limit
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return detailed, pagination, nil
}

func GetUserDisplayName(userID string) (string, error) {

	var firstName, lastName, email, phone sql.NullString

	query := `
		SELECT first_name, last_name, email, phone_number
		FROM users
		WHERE user_id = ?
	`

	err := DB.QueryRow(query, userID).Scan(&firstName, &lastName, &email, &phone)
	if err != nil {
		return "", err
	}
	// First + Last Name
	if firstName.Valid && lastName.Valid {
		return strings.TrimSpace(firstName.String + " " + lastName.String), nil
	}

	// Email fallback
	if email.Valid {
		return email.String, nil
	}

	// Phone fallback
	if phone.Valid {
		return phone.String, nil
	}

	return "Unknown User", nil
}

func AddNewReview(req dtos.ReviewRequest, productID string) (dtos.ReviewResponse, error) {
	// Step 1: Check if user has purchased the product with a COLLECTED order
	var purchaseCount int
	purchaseQuery := `
		SELECT COUNT(*) 
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE oi.product_id = ? AND o.user_id = ? AND (LOWER(o.status) = LOWER('delivered') OR LOWER(o.status) = LOWER('completed'))
	`
	err := DB.QueryRow(purchaseQuery, productID, req.UserID).Scan(&purchaseCount)

	if err != nil {
		return dtos.ReviewResponse{}, fmt.Errorf("failed to check product purchase: %v", err)
	}
	if purchaseCount == 0 {
		return dtos.ReviewResponse{}, fmt.Errorf("user has not purchased this product or order not delivered")
	}

	// Step 2: Check if the user already left a review for this product
	var reviewCount int
	reviewCheckQuery := `
		SELECT COUNT(*) 
		FROM product_reviews 
		WHERE product_id = ? AND user_id = ?
	`
	err = DB.QueryRow(reviewCheckQuery, productID, req.UserID).Scan(&reviewCount)
	if err != nil {
		return dtos.ReviewResponse{}, fmt.Errorf("failed to check existing review: %v", err)
	}
	if reviewCount > 0 {
		return dtos.ReviewResponse{}, fmt.Errorf("user has already reviewed this product")
	}

	// Step 3: Insert the new review
	reviewID, _ := shortid.Generate()
	insertQuery := `
		INSERT INTO product_reviews (review_id, product_id, user_id, score, details)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(insertQuery, reviewID, productID, req.UserID, req.Score, req.Details)
	if err != nil {
		return dtos.ReviewResponse{}, fmt.Errorf("failed to insert review for user %s: %v", req.UserID, err)
	}

	// Step 4: Return the response
	user, err := GetUserDisplayName(req.UserID)
	if err != nil {
		return dtos.ReviewResponse{}, err
	}
	res := dtos.ReviewResponse{
		ID:        reviewID,
		User:      user,
		Score:     req.Score,
		Details:   req.Details,
		CreatedAt: time.Now(),
	}

	return res, nil
}

func UpdateReview(req dtos.UpdateReview, reviewID string) error {
	exists, err := RecordExists("product_reviews", "review_id = ?", reviewID)
	if err != nil {
		return fmt.Errorf("failed to check variant existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("review not found")
	}
	query := "UPDATE product_reviews SET"
	args := []interface{}{}
	updates := []string{}

	if req.Status != "" {
		updates = append(updates, "status = ?")
		args = append(args, req.Status)
	}
	if req.Details != "" {
		updates = append(updates, "details = ?")
		args = append(args, req.Details)
	}
	if req.Score != 0 {
		updates = append(updates, "score = ?")
		args = append(args, req.Score)
	}

	if len(updates) == 0 {
		return nil // Nothing to update
	}

	query += " " + strings.Join(updates, ", ") + " WHERE review_id = ?"
	args = append(args, reviewID)

	if _, err := DB.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update review: %v", err)
	}

	return nil
}
func DeleteReview(reviewID string) error {
	// Check if review exists
	exists, err := RecordExists("product_reviews", "review_id = ?", reviewID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("review not found")
	}
	_, err = DB.Exec("DELETE FROM product_reviews WHERE review_id = ?", reviewID)
	return err
}
