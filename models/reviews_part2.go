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
	"database/sql"
	"ekomasi_backend/dtos"
	"fmt"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

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
