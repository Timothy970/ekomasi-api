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
