package models

import (
	"adenzo_backend/dtos"
	"fmt"
	"strings"

	"github.com/teris-io/shortid"
)

func GetProductReviews(productID, reviewID string, limit, page int) ([]dtos.ReviewResponse, *dtos.PaginationMeta, error) {
	if reviewID != "" {
		query := `
			SELECT review_id, product_id, user_id, score, details, created_at
			FROM product_reviews
			WHERE product_id = ? AND review_id = ? AND status = 'approved'
		`
		row := DB.QueryRow(query, productID, reviewID)

		var r dtos.ReviewResponse
		err := row.Scan(&r.ID, &r.ProductID, &r.UserID, &r.Score, &r.Details, &r.CreatedAt)
		if err != nil {
			return nil, nil, err
		}

		return []dtos.ReviewResponse{r}, nil, nil
	}

	// When no reviewID, apply pagination
	offset := (page - 1) * limit

	// Count total reviews
	var totalItems int
	countQuery := `SELECT COUNT(*) FROM product_reviews WHERE product_id = ? AND status = 'approved'`
	err := DB.QueryRow(countQuery, productID).Scan(&totalItems)
	if err != nil {
		return nil, nil, err
	}

	// Fetch paginated reviews
	query := `
		SELECT review_id, product_id, user_id, score, details, created_at
		FROM product_reviews
		WHERE product_id = ? AND status = 'approved'
		LIMIT ? OFFSET ?
	`
	rows, err := DB.Query(query, productID, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var reviews []dtos.ReviewResponse
	for rows.Next() {
		var r dtos.ReviewResponse
		if err := rows.Scan(&r.ID, &r.ProductID, &r.UserID, &r.Score, &r.Details, &r.CreatedAt); err != nil {
			return nil, nil, err
		}
		reviews = append(reviews, r)
	}

	// Build pagination metadata
	totalPages := (totalItems + limit - 1) / limit
	pagination := &dtos.PaginationMeta{
		Page:       page,
		Size:       limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return reviews, pagination, nil
}

func AddNewReview(req dtos.ReviewRequest, productID string) (dtos.ReviewResponse, error) {
	// Step 1: Check if user has purchased the product with a COLLECTED order
	var purchaseCount int
	purchaseQuery := `
		SELECT COUNT(*) 
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.order_id
		WHERE oi.product_id = ? AND o.user_id = ? AND o.status = 'COLLECTED'
	`
	err := DB.QueryRow(purchaseQuery, productID, req.UserID).Scan(&purchaseCount)
	if err != nil {
		return dtos.ReviewResponse{}, fmt.Errorf("failed to check product purchase: %v", err)
	}
	if purchaseCount == 0 {
		return dtos.ReviewResponse{}, fmt.Errorf("user has not purchased this product or order not collected")
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
	res := dtos.ReviewResponse{
		ID:        reviewID,
		ProductID: productID,
		UserID:    req.UserID,
		Score:     req.Score,
		Details:   req.Details,
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
