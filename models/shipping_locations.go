package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
)

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
