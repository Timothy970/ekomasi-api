// Package models provides data access functions for the Ekomasi e-commerce backend.
//
// reviews.go handles product review management including:
//   - Review submission with purchase verification
//   - Review retrieval with pagination and filtering
//   - Review analytics (average score, rating distribution)
//   - Review moderation (update, delete)
//   - User display name resolution
package models

import (
	"ekomasi_backend/dtos"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

// isReviewThere validates that a review exists in the database.
//
// Parameters:
//   - reviewID: string - The review_id to validate
//
// Returns:
//   - error: "review not found", database error, or nil if review exists
func isReviewThere(db DBExecutor, reviewID string) error {
	exists, err := RecordExists(db, "product_reviews", "review_id = ?", reviewID)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("review not found")
	}
	return nil
}

// GetProductReview retrieves a single specific review for a product.
//
// This function validates product and review existence, then returns the review
// with user display name resolved.
//
// Parameters:
//   - productID: string - The product to retrieve review for
//   - reviewID: string - The specific review to retrieve
//   - limit: int - Unused (legacy parameter, kept for API compatibility)
//   - page: int - Unused (legacy parameter, kept for API compatibility)
//
// Returns:
//   - []dtos.ReviewResponse: Single-item array with review details
//   - *dtos.PaginationMeta: nil (no pagination for single review)
//   - error: "product not found", "review not found", "review for the product not found", or nil on success
func GetProductReview(db DBExecutor, productID, reviewID string, limit, page int) ([]dtos.ReviewResponse, *dtos.PaginationMeta, error) {
	// Validate product exists
	err := IsProductThere(db, productID)
	if err != nil {
		return nil, nil, err
	}

	// Validate review exists
	err = isReviewThere(db, reviewID)
	if err != nil {
		return nil, nil, err
	}

	// Retrieve review for specific product and review ID
	query := `
			SELECT review_id, user_id, score, details, created_at
			FROM product_reviews
			WHERE product_id = ? AND review_id = ?
			LIMIT 1
		`

	var r dtos.ReviewResponse
	var userID string
	err = db.QueryRow(query, productID, reviewID).
		Scan(&r.ID, &userID, &r.Score, &r.Details, &r.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("review for the product not found")
		}
		return nil, nil, err
	}

	// Resolve user display name (first+last name, email, phone, or "Unknown User")
	r.User, err = GetUserDisplayName(db, userID)
	if err != nil {
		return nil, nil, err
	}
	return []dtos.ReviewResponse{r}, nil, nil
}

// GetProductReviews retrieves all reviews for a product with analytics and pagination.
//
// This comprehensive function provides:
//   - Paginated review list with sorting
//   - Average score calculation
//   - Rating distribution (1-5 stars with counts)
//   - Optional filtering by rating
//
// Parameters:
//   - productID: string - Product to retrieve reviews for
//   - sortBy: string - Sort order: "newest", "oldest", "highest", "lowest" (default: newest)
//   - rating: int - Filter by specific score (1-5). 0 = no filter
//   - limit: int - Reviews per page
//   - page: int - Page number (1-indexed)
//
// Returns:
//   - *dtos.DetailedReviewResponse: Contains:
//   - Reviews: Array of review objects with user display names
//   - AverageScore: Average rating (0.0 if no reviews)
//   - ScoreCounts: Array of 5 items (scores 1-5) with counts (missing = 0)
//   - *dtos.PaginationMeta: Page info (page, size, totals, navigation flags)
//   - error: Database error or nil on success
func GetProductReviews(db DBExecutor, productID, sortBy string, rating, limit, page int) (*dtos.DetailedReviewResponse, *dtos.PaginationMeta, error) {
	offset := (page - 1) * limit

	totalItems, err := countReviews(db, productID, rating)
	if err != nil {
		return nil, nil, err
	}

	reviewList, err := fetchPaginatedReviews(db, productID, sortBy, rating, limit, offset)
	if err != nil {
		return nil, nil, err
	}

	finalScoreCounts, averageScore, err := calculateScoreAnalytics(db, productID, totalItems)
	if err != nil {
		return nil, nil, err
	}

	detailed := &dtos.DetailedReviewResponse{
		Reviews:      reviewList,
		AverageScore: averageScore,
		ScoreCounts:  finalScoreCounts,
	}
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

// countReviews counts total reviews matching the filter criteria
func countReviews(db DBExecutor, productID string, rating int) (int, error) {
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
	err := db.QueryRow(countQuery, countArgs...).Scan(&totalItems)
	return totalItems, err
}

// fetchPaginatedReviews retrieves reviews with sorting and pagination
func fetchPaginatedReviews(db DBExecutor, productID, sortBy string, rating, limit, offset int) ([]dtos.ReviewResponse, error) {
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

	query += getSortClauseData(sortBy)
	query += " LIMIT ? OFFSET ?"
	queryArgs = append(queryArgs, limit, offset)

	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewList []dtos.ReviewResponse
	for rows.Next() {
		var r dtos.ReviewResponse
		var userID string
		if err := rows.Scan(&r.ID, &userID, &r.Score, &r.Details, &r.CreatedAt); err != nil {
			return nil, err
		}

		r.User, err = GetUserDisplayName(db, userID)
		if err != nil {
			return nil, err
		}
		reviewList = append(reviewList, r)
	}

	return reviewList, nil
}

// getSortClause returns the SQL ORDER BY clause based on sort preference
func getSortClauseData(sortBy string) string {
	switch strings.ToLower(sortBy) {
	case "newest":
		return " ORDER BY created_at DESC"
	case "oldest":
		return " ORDER BY created_at ASC"
	case "highest":
		return " ORDER BY score DESC"
	case "lowest":
		return " ORDER BY score ASC"
	default:
		return " ORDER BY created_at DESC"
	}
}

// calculateScoreAnalytics computes rating distribution and average score
func calculateScoreAnalytics(db DBExecutor, productID string, totalItems int) ([]dtos.ScoreCount, float64, error) {
	scoreQuery := `
		SELECT score, COUNT(*)
		FROM product_reviews
		WHERE product_id = ?
		GROUP BY score
	`

	scoreRows, err := db.Query(scoreQuery, productID)
	if err != nil {
		return nil, 0, err
	}
	defer scoreRows.Close()

	scoreMap := make(map[int]int)
	totalScore := 0

	for scoreRows.Next() {
		var score, count int
		if err := scoreRows.Scan(&score, &count); err != nil {
			return nil, 0, err
		}
		scoreMap[score] = count
		totalScore += score * count
	}

	finalScoreCounts := make([]dtos.ScoreCount, 0, 5)
	for i := 1; i <= 5; i++ {
		finalScoreCounts = append(finalScoreCounts, dtos.ScoreCount{
			Score: i,
			Count: scoreMap[i],
		})
	}

	averageScore := 0.0
	if totalItems > 0 {
		averageScore = float64(totalScore) / float64(totalItems)
	}

	return finalScoreCounts, averageScore, nil
}

// GetUserDisplayName resolves a user ID to a display-friendly name.
//
// This function follows a fallback strategy:
//  1. First name + Last name (if both available)
//  2. Email address (if available)
//  3. Phone number (if available)
//  4. "Unknown User" (if no identifiable info)
//
// Parameters:
//   - userID: string - The user_id to resolve
//
// Returns:
//   - string: Display name following fallback hierarchy
//   - error: Database error or nil on success
func GetUserDisplayName(db DBExecutor, userID string) (string, error) {

	var firstName, lastName, email, phone sql.NullString

	query := `
		SELECT first_name, last_name, email, phone_number
		FROM users
		WHERE user_id = ?
	`

	err := db.QueryRow(query, userID).Scan(&firstName, &lastName, &email, &phone)
	if err != nil {
		return "", err
	}

	// Priority 1: First + Last Name
	if firstName.Valid && lastName.Valid {
		return strings.TrimSpace(firstName.String + " " + lastName.String), nil
	}

	// Priority 2: Email fallback
	if email.Valid {
		return email.String, nil
	}

	// Priority 3: Phone fallback
	if phone.Valid {
		return phone.String, nil
	}

	// Priority 4: Default fallback
	return "Unknown User", nil
}

// AddNewReview creates a new product review with validation.
//
// This function enforces review policies:
//  1. User must have purchased the product (order status = "delivered" or "completed")
//  2. User can only review a product once
//
// Parameters:
//   - req: dtos.ReviewRequest containing:
//   - UserID: User submitting the review
//   - Score: Rating (1-5 stars)
//   - Details: Review text/description
//   - productID: string - Product being reviewed
//
// Returns:
//   - dtos.ReviewResponse: Created review with:
//   - ID: Generated review_id
//   - User: Display name
//   - Score, Details: From request
//   - CreatedAt: Current timestamp
//   - error: "user has not purchased this product or order not delivered",
//     "user has already reviewed this product", database error, or nil on success
func AddNewReview(db DBExecutor, req dtos.ReviewRequest, productID string) (dtos.ReviewResponse, error) {
	// Step 1: Verify user purchased the product with a delivered/completed order
	var purchaseCount int
	purchaseQuery := `
		SELECT COUNT(*) 
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE oi.product_id = ? AND o.user_id = ? AND (LOWER(o.status) = LOWER('delivered') OR LOWER(o.status) = LOWER('completed'))
	`
	err := db.QueryRow(purchaseQuery, productID, req.UserID).Scan(&purchaseCount)

	if err != nil {
		return dtos.ReviewResponse{}, fmt.Errorf("failed to check product purchase: %v", err)
	}
	if purchaseCount == 0 {
		return dtos.ReviewResponse{}, fmt.Errorf("user has not purchased this product or order not delivered")
	}

	// Step 2: Ensure user hasn't already reviewed this product
	var reviewCount int
	reviewCheckQuery := `
		SELECT COUNT(*) 
		FROM product_reviews 
		WHERE product_id = ? AND user_id = ?
	`
	err = db.QueryRow(reviewCheckQuery, productID, req.UserID).Scan(&reviewCount)
	if err != nil {
		return dtos.ReviewResponse{}, fmt.Errorf("failed to check existing review: %v", err)
	}
	if reviewCount > 0 {
		return dtos.ReviewResponse{}, fmt.Errorf("user has already reviewed this product")
	}

	// Step 3: Insert the new review with generated ID
	reviewID, _ := shortid.Generate()
	insertQuery := `
		INSERT INTO product_reviews (review_id, product_id, user_id, score, details)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = db.Exec(insertQuery, reviewID, productID, req.UserID, req.Score, req.Details)
	if err != nil {
		return dtos.ReviewResponse{}, fmt.Errorf("failed to insert review for user %s: %v", req.UserID, err)
	}

	// Step 4: Build and return the response with user display name
	user, err := GetUserDisplayName(db, req.UserID)
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

// UpdateReview updates an existing review with validation.
//
// This function validates:
//  1. Review exists
//  2. Product exists
//  3. Review belongs to the specified product
//
// Supports partial updates (only provided fields are updated).
//
// Parameters:
//   - req: dtos.UpdateReview containing optional fields:
//   - Status: *string - Review status (e.g., "approved", "flagged")
//   - Details: string - Updated review text (empty = no change)
//   - Score: int - Updated rating (0 = no change)
//   - reviewID: string - The review to update
//   - productID: string - Product for validation
//
// Returns:
//   - error: "review not found", "product not found", "review does not belong to the specified product",
//     database error, or nil on success
func UpdateReview(db DBExecutor, req dtos.UpdateReview, reviewID string, productID string) error {
	// Validate review exists
	err := isReviewThere(db, reviewID)
	if err != nil {
		return err
	}

	// Validate product exists
	err = IsProductThere(db, productID)
	if err != nil {
		return err
	}

	// Verify review belongs to the product
	var existingProductID string
	reviewCheckQuery := "SELECT product_id FROM product_reviews WHERE review_id = ?"
	err = db.QueryRow(reviewCheckQuery, reviewID).Scan(&existingProductID)
	if err != nil {
		return fmt.Errorf("failed to fetch review: %v", err)
	}
	if existingProductID != productID {
		return fmt.Errorf("review does not belong to the specified product")
	}

	// Build dynamic UPDATE query with only provided fields
	query := "UPDATE product_reviews SET"
	args := []interface{}{}
	updates := []string{}

	if req.Status != nil {
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
		return nil // Nothing to update - idempotent
	}

	// Construct and execute the update query
	query += " " + strings.Join(updates, ", ") + " WHERE review_id = ?"
	args = append(args, reviewID)

	if _, err := db.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update review: %v", err)
	}

	return nil
}

// DeleteReview permanently removes a review from the database.
//
// Warning: This is a hard delete. Consider soft delete (status update) to maintain
//
//	review history for auditing and analytics.
//
// Parameters:
//   - reviewID: string - The review_id to delete
//
// Returns:
//   - error: "review not found", database error, or nil on success
func DeleteReview(db DBExecutor, reviewID string) error {
	// Validate review exists before deletion
	exists, err := RecordExists(db, "product_reviews", "review_id = ?", reviewID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("review not found")
	}

	// Hard delete the review
	_, err = db.Exec("DELETE FROM product_reviews WHERE review_id = ?", reviewID)
	return err
}
