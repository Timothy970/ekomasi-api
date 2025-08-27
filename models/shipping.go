package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"errors"
	"fmt"

	"github.com/teris-io/shortid"
)

var nolocation = "location not found"

func GetDeliveryRate(location string) (float64, string, error) {
	charge, dbResult := 0.0, ""
	query := `
		SELECT location, charge 
		FROM delivery_rates 
		WHERE LOWER(location) LIKE CONCAT('%', ?, '%')
		ORDER BY LOCATE(?, LOWER(location)) ASC
		LIMIT 1
	`
	err := DB.QueryRow(query, location, location).Scan(&dbResult, &charge)
	if err != nil {
		if err == sql.ErrNoRows {
			return 2000.00, location, nil // not found, to do:better way to get the amount
		}
		return 0, "", err
	}
	return charge, dbResult, nil
}

func DeleteDeliveryFeedback(deliveryFeedbackID string) error {
	exists, err := RecordExists("delivery_feedback", "delivery_feedback_id = ?", deliveryFeedbackID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("delivery feedback not found")
	}
	query := `
		DELETE FROM delivery_feedback
		WHERE delivery_feedback_id =?
	`
	_, err = DB.Exec(query, deliveryFeedbackID)
	return err
}
func isDeliveryThere(id string) error {
	exists, err := RecordExists("deliveries", "delivery_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed : %w", err)
	}
	if !exists {
		return fmt.Errorf("delivery not found")
	}
	return nil
}
func AddNewDeliveryFeedback(req dtos.DeliveryFeedback) error {
	err := isDeliveryThere(req.DeliveryID)
	if err != nil {
		return err
	}

	deliveryFeedbackID, _ := shortid.Generate()

	// Step 1: Execute INSERT
	query := `
		INSERT INTO delivery_feedback (delivery_feedback_id, delivery_id, score, details)
		VALUES (?, ?, ?, ?)
	`
	_, err = DB.Exec(query, deliveryFeedbackID, req.DeliveryID, req.Score, req.Details)
	if err != nil {
		return fmt.Errorf("failed to insert delivery feed back : %v", err)
	}

	return nil
}
func GetDeliveryFeedBack(deliveryID string) ([]dtos.DeliveryFeedback, error) {
	er := isDeliveryThere(deliveryID)
	if er != nil {
		return nil, er
	}
	query := `
		SELECT delivery_feedback_id, delivery_id, score, details
		FROM delivery_feedback
	`
	var rows *sql.Rows
	var err error

	// if feedbackID == "" {
	// 	rows, err = DB.Query(query)
	// } else {
	query += ` WHERE delivery_id = ?`
	rows, err = DB.Query(query, deliveryID)
	// }
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

func GetDeliveryUserFeedBack(userID string) ([]dtos.DeliveryFeedback, error) {
	er := isUserThere(userID)
	if er != nil {
		return nil, er
	}
	query := `
		SELECT df.delivery_feedback_id, df.delivery_id, df.score, df.details
		FROM delivery_feedback df
		JOIN deliveries d ON df.delivery_id = d.delivery_id
		JOIN orders o ON d.order_id = o.order_id
		WHERE o.user_id = ?
	`

	rows, err := DB.Query(query, userID)
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

func AddNewShippingRate(req dtos.ShippingCostResponse) error {
	if req.Charge == 0 {
		req.Charge = 0.0
	}

	var existing int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_rates WHERE LOWER(location) = LOWER(?)",
		req.Location,
	).Scan(&existing)
	if err != nil {
		return fmt.Errorf("failed to check location existence: %v", err)
	}

	if existing > 0 {
		// Update instead of insert
		_, err := DB.Exec("UPDATE delivery_rates SET charge = ? WHERE location = ?", req.Charge, req.Location)
		if err != nil {
			return fmt.Errorf("failed to update shipping rate: %v", err)
		}
		return nil
	}

	// 2. Insert new location
	_, err = DB.Exec("INSERT INTO delivery_rates (location, charge) VALUES (?, ?)", req.Location, req.Charge)
	if err != nil {
		return fmt.Errorf("failed to insert shipping rate: %v", err)
	}

	return nil
}

// List locations with pagination
func ListLocations(page, size int) ([]dtos.Location, dtos.PaginationMeta, error) {
	var locations []dtos.Location
	var meta dtos.PaginationMeta

	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	// Get total count
	var total int
	err := DB.QueryRow("SELECT COUNT(*) FROM delivery_rates").Scan(&total)
	if err != nil {
		return nil, meta, err
	}

	offset := (page - 1) * size
	rows, err := DB.Query("SELECT id, location, charge FROM delivery_rates LIMIT ? OFFSET ?", size, offset)
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

	totalPages := (total + size - 1) / size
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

// Get location by ID
func GetLocationByID(id string) (dtos.Location, error) {
	var loc dtos.Location
	err := DB.QueryRow("SELECT id, location, charge FROM delivery_rates WHERE id = ?", id).Scan(&loc.ID, &loc.Location, &loc.Charge)
	if err != nil {
		if err == sql.ErrNoRows {
			return loc, errors.New(nolocation)
		}
		return loc, err
	}
	return loc, nil
}

// Update location
func UpdateLocation(loc dtos.UpdateLocation, locationID string) error {
	exists, err := RecordExists("delivery_rates", "id = ?", locationID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(nolocation)
	}
	stmt, err := DB.Prepare("UPDATE delivery_rates SET location = ?, charge = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(loc.Location, loc.Charge, locationID)
	return err
}

// Delete location
func DeleteLocation(id string) error {
	exists, err := RecordExists("delivery_rates", "id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(nolocation)
	}
	stmt, err := DB.Prepare("DELETE FROM delivery_rates WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	return err
}
