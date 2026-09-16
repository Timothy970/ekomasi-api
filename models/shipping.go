// Package models provides data access functions for the Ekomasi e-commerce backend.
//
// shipping.go handles shipping and delivery management including:
//   - Delivery rate calculation by location
//   - Delivery feedback collection and retrieval
//   - Shipping rate management (CRUD)
//   - Location-based delivery cost lookup
package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"fmt"

	"github.com/teris-io/shortid"
)

// nolocation is the standard error message for location not found errors
var nolocation = "location not found"

// GetDeliveryRate retrieves the shipping cost for a given location.
//
// This function performs fuzzy location matching using LIKE and returns the
// best match based on location string position.
//
// Parameters:
//   - location: string - The delivery location to search for (partial match supported)
//
// Returns:
//   - float64: Delivery charge (2000.00 default if location not found)
//   - string: Matched location name from database (or input location if not found)
//   - error: Database error or nil on success
func GetDeliveryRate(db DBExecutor, location string) (float64, string, error) {
	charge, dbResult := 0.0, ""

	// Fuzzy match location using LIKE with case-insensitive search
	// Order by position of match for best result
	query := `
		SELECT location, charge 
		FROM delivery_rates 
		WHERE LOWER(location) LIKE CONCAT('%', ?, '%')
		ORDER BY LOCATE(?, LOWER(location)) ASC
		LIMIT 1
	`
	err := db.QueryRow(query, location, location).Scan(&dbResult, &charge)
	if err != nil {
		if err == sql.ErrNoRows {
			// Location not found - return default charge (TODO1: better way to get the amount)
			return 2000.00, location, nil
		}
		return 0, "", err
	}
	return charge, dbResult, nil
}

// DeleteDeliveryFeedback permanently removes delivery feedback.
//
// Parameters:
//   - deliveryFeedbackID: string - The delivery_feedback_id to delete
//
// Returns:
//   - error: "delivery feedback not found", database error, or nil on success
func DeleteDeliveryFeedback(db DBExecutor, deliveryFeedbackID string) error {
	// Validate feedback exists
	exists, err := RecordExists(db, "delivery_feedback", "delivery_feedback_id = ?", deliveryFeedbackID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("delivery feedback not found")
	}

	// Delete feedback record
	query := `
		DELETE FROM delivery_feedback
		WHERE delivery_feedback_id =?
	`
	_, err = db.Exec(query, deliveryFeedbackID)
	return err
}

// isDeliveryThere validates that a delivery exists in the database.
//
// Parameters:
//   - id: string - The delivery_id to validate
//
// Returns:
//   - error: "delivery not found", database error, or nil if delivery exists
func isDeliveryThere(db DBExecutor, id string) error {
	exists, err := RecordExists(db, "deliveries", "delivery_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed : %w", err)
	}
	if !exists {
		return fmt.Errorf("delivery not found")
	}
	return nil
}

// AddNewDeliveryFeedback creates new delivery feedback.
//
// This function allows customers to rate and review their delivery experience.
//
// Parameters:
//   - req: dtos.DeliveryFeedback containing:
//   - DeliveryID: Delivery to provide feedback for
//   - Score: Rating score
//   - Details: Feedback text/description
//
// Returns:
//   - error: "delivery not found", "failed to insert delivery feed back", or nil on success
func AddNewDeliveryFeedback(db DBExecutor, req dtos.DeliveryFeedback) error {
	// Validate delivery exists
	err := isDeliveryThere(db, req.DeliveryID)
	if err != nil {
		return err
	}

	deliveryFeedbackID, _ := shortid.Generate()

	// Insert feedback with generated ID
	query := `
		INSERT INTO delivery_feedback (delivery_feedback_id, delivery_id, score, details)
		VALUES (?, ?, ?, ?)
	`
	_, err = db.Exec(query, deliveryFeedbackID, req.DeliveryID, req.Score, req.Details)
	if err != nil {
		return fmt.Errorf("failed to insert delivery feed back : %v", err)
	}

	return nil
}

// GetDeliveryFeedBack retrieves all feedback for a specific delivery.
//
// Parameters:
//   - deliveryID: string - The delivery_id to get feedback for
//
// Returns:
//   - []dtos.DeliveryFeedback: Array of feedback containing FeedbackID, DeliveryID, Score, Details
//   - error: "delivery not found", database error, or nil on success
func GetDeliveryFeedBack(db DBExecutor, deliveryID string) ([]dtos.DeliveryFeedback, error) {
	// Validate delivery exists
	er := isDeliveryThere(db, deliveryID)
	if er != nil {
		return nil, er
	}

	// Query feedback for specific delivery
	query := `
		SELECT delivery_feedback_id, delivery_id, score, details
		FROM delivery_feedback
	`
	var rows *sql.Rows
	var err error

	query += ` WHERE delivery_id = ?`
	rows, err = db.Query(query, deliveryID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feedback []dtos.DeliveryFeedback
	for rows.Next() {
		var r dtos.DeliveryFeedback
		if err := rows.Scan(&r.FeedbackID, &r.DeliveryID, &r.Score, &r.Details); err != nil {
			return nil, err
		}
		feedback = append(feedback, r)
	}

	return feedback, nil
}

// GetDeliveryUserFeedBack retrieves all delivery feedback for a specific user.
//
// This function joins across deliveries and orders to get all feedback
// for deliveries belonging to the user's orders.
//
// Parameters:
//   - userID: string - The user_id to get feedback for
//
// Returns:
//   - []dtos.DeliveryFeedback: Array of feedback for user's deliveries
//   - error: "user not found", database error, or nil on success
func GetDeliveryUserFeedBack(db DBExecutor, userID string) ([]dtos.DeliveryFeedback, error) {
	// Validate user exists
	er := isUserThere(db, userID)
	if er != nil {
		return nil, er
	}

	// Join through deliveries and orders to get user's feedback
	query := `
		SELECT df.delivery_feedback_id, df.delivery_id, df.score, df.details
		FROM delivery_feedback df
		JOIN deliveries d ON df.delivery_id = d.delivery_id
		JOIN orders o ON d.order_id = o.order_id
		WHERE o.user_id = ?
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feedback []dtos.DeliveryFeedback
	for rows.Next() {
		var r dtos.DeliveryFeedback
		if err := rows.Scan(&r.FeedbackID, &r.DeliveryID, &r.Score, &r.Details); err != nil {
			return nil, err
		}
		feedback = append(feedback, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return feedback, nil
}

// AddNewShippingRate creates or updates a shipping rate for a location.
//
// This function implements upsert logic:
//   - If location exists: Updates the charge
//   - If location doesn't exist: Inserts new record
//
// Parameters:
//   - req: dtos.ShippingCostResponse containing:
//   - Location: Delivery location name
//   - Charge: Delivery cost (defaults to 0.0 if not provided)
//
// Returns:
//   - error: "failed to check location existence", "failed to update shipping rate",
//     "failed to insert shipping rate", or nil on success
func AddNewShippingRate(db DBExecutor, req dtos.ShippingCostResponse) error {
	// Default charge to 0.0 if not provided
	if req.Charge == 0 {
		req.Charge = 0.0
	}

	// Check if location already exists (case-insensitive)
	var existing int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM delivery_rates WHERE LOWER(location) = LOWER(?)",
		req.Location,
	).Scan(&existing)
	if err != nil {
		return fmt.Errorf("failed to check location existence: %v", err)
	}

	if existing > 0 {
		// Location exists - update charge
		_, err := db.Exec("UPDATE delivery_rates SET charge = ? WHERE location = ?", req.Charge, req.Location)
		if err != nil {
			return fmt.Errorf("failed to update shipping rate: %v", err)
		}
		return nil
	}

	// Location doesn't exist - insert new record
	_, err = db.Exec("INSERT INTO delivery_rates (location, charge) VALUES (?, ?)", req.Location, req.Charge)
	if err != nil {
		return fmt.Errorf("failed to insert shipping rate: %v", err)
	}

	return nil
}

// ListLocations retrieves delivery locations with pagination.
//
// Parameters:
//   - page: int - Page number (minimum 1, defaults to 1 if < 1)
//   - size: int - Items per page (minimum 1, defaults to 10 if < 1)
//
// Returns:
//   - []dtos.Location: Array of locations with ID, Location, Charge
//   - dtos.PaginationMeta: Pagination info (page, size, totals, navigation flags)
//   - error: Database error or nil on success
func ListLocations(db DBExecutor, page, size int) ([]dtos.Location, dtos.PaginationMeta, error) {
	var locations []dtos.Location
	var meta dtos.PaginationMeta

	// Validate and set defaults for pagination parameters
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	// Get total count for pagination calculation
	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM delivery_rates").Scan(&total)
	if err != nil {
		return nil, meta, err
	}

	// Fetch paginated locations
	offset := (page - 1) * size
	rows, err := db.Query("SELECT id, location, charge FROM delivery_rates LIMIT ? OFFSET ?", size, offset)
	if err != nil {
		return nil, meta, err
	}
	defer rows.Close()

	for rows.Next() {
		var loc dtos.Location
		if err := rows.Scan(&loc.ID, &loc.Location, &loc.Charge); err != nil {
			return nil, meta, err
		}
		locations = append(locations, loc)
	}

	// Build pagination metadata
	totalPages := (total + size - 1) / size // Ceiling division
	meta = dtos.PaginationMeta{
		Page:       page,
		Size:       size,
		TotalItems: total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}

	return locations, meta, nil
}

// GetLocationByID retrieves a single delivery location by its ID.
//
// Parameters:
//   - id: int - The location ID to retrieve
//
// Returns:
//   - dtos.Location: Location details (ID, Location, Charge)
//   - error: "location not found", database error, or nil on success
func GetLocationByID(db DBExecutor, id int) (dtos.Location, error) {
	var loc dtos.Location
	err := db.QueryRow("SELECT id, location, charge FROM delivery_rates WHERE id = ?", id).Scan(&loc.ID, &loc.Location, &loc.Charge)
	if err != nil {
		if err == sql.ErrNoRows {
			return loc, errors.New(nolocation)
		}
		return loc, err
	}
	return loc, nil
}

// UpdateLocation updates an existing delivery location's details.
//
// Parameters:
//   - loc: dtos.UpdateLocation containing:
//   - Location: New location name
//   - Charge: New delivery charge
//   - locationID: string - The location ID to update
//
// Returns:
//   - error: "location not found", database error, or nil on success
func UpdateLocation(db DBExecutor, loc dtos.UpdateLocation, locationID string) error {
	// Validate location exists
	exists, err := RecordExists(db, "delivery_rates", "id = ?", locationID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(nolocation)
	}

	// Update location details
	_, err = db.Exec("UPDATE delivery_rates SET location = ?, charge = ? WHERE id = ?", loc.Location, loc.Charge, locationID)

	return err
}

// DeleteLocation permanently removes a delivery location.
//
// Parameters:
//   - id: string - The location ID to delete
//
// Returns:
//   - error: "location not found", database error, or nil on success
func DeleteLocation(db DBExecutor, id string) error {
	// Validate location exists
	exists, err := RecordExists(db, "delivery_rates", "id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(nolocation)
	}

	// Delete location record
	_, err = db.Exec("DELETE FROM delivery_rates WHERE id = ?", id)

	return err
}
