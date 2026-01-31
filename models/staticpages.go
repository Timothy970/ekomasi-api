// Package models provides data access functions for the Adenzo e-commerce platform.
//
// This file contains functions for managing static pages (CMS content):
//   - Create, retrieve, update, delete static pages
//   - Title and path uniqueness validation
//   - JSON section data management
//   - Author information enrichment
//   - Dynamic query filtering and ordering
package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

// CreateStaticPage creates a new static page in the CMS.
//
// This function validates title and path uniqueness, generates a unique ID,
// marshals section data to JSON, and inserts the page into the database.
//
// Parameters:
//   - req: dtos.StaticPageRequest containing:
//   - Title: Page title (must be unique, case-insensitive)
//   - Description: Page description/summary
//   - Path: URL path (must be unique, case-insensitive)
//   - Sections: Array of page sections (marshaled to JSON)
//   - userID: string - ID of the user creating the page (author)
//
// Returns:
//   - error: "static page with this title already exists",
//     "static page with this path already exists",
//     database error, or nil on success
func CreateStaticPage(req dtos.StaticPageRequest, userID string) error {
	// Validate title and path uniqueness (case-insensitive)
	err := isStaticPageThereByTitleOrPath(req.Title, req.Path)
	if err != nil {
		return err
	}

	// Generate unique page ID
	staticPageID, _ := shortid.Generate()

	// Marshal sections to JSON for storage
	data, _ := json.Marshal(req.Sections)

	// Insert new static page
	query := `
		INSERT INTO static_pages (static_page_id, title, description, data, path, user_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(query, staticPageID, req.Title, req.Description, data, req.Path, userID)
	return err
}

// isStaticPageThereByTitleOrPath validates uniqueness of static page title and path.
//
// This helper function performs case-insensitive checks to ensure no duplicate
// titles or paths exist in the system.
//
// Parameters:
//   - title: string - Page title to check for uniqueness
//   - path: string - URL path to check for uniqueness
//
// Returns:
//   - error: "static page with this title already exists",
//     "static page with this path already exists",
//     database error, or nil if both are unique
func isStaticPageThereByTitleOrPath(title, path string) error {
	// Check if title already exists (case-insensitive)
	exists, err := RecordExists(DB, "static_pages", "LOWER(title) = LOWER(?)", title)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("static page with this title already exists")
	}

	// Check if path already exists (case-insensitive)
	exists, err = RecordExists(DB, "static_pages", "LOWER(path) = LOWER(?)", path)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("static page with this path already exists")
	}

	return nil
}

// GetStaticPages retrieves static pages with optional title filtering.
//
// This function supports dynamic query building with case-insensitive title search.
// Results are ordered by creation date (newest first) and include enriched author info.
//
// Parameters:
//   - query: string - Optional title to filter by (case-insensitive exact match).
//     Empty string returns all pages.
//
// Returns:
//   - []dtos.StaticPageRequest: Array of static pages with:
//   - StaticPageID, Title, Description, Path
//   - Sections: Unmarshaled JSON section data
//   - CreatedAt, UpdatedAt: Formatted datetime strings
//   - Author: Display name of the page creator
//   - error: Database error, unmarshal error, or nil on success
func GetStaticPages(query string) ([]dtos.StaticPageRequest, error) {
	var (
		staticPages []dtos.StaticPageRequest
		args        []interface{}
	)

	// Build base query
	pageQuery := `
		SELECT static_page_id, title, description, path, data, created_at, updated_at, user_id
		FROM static_pages
	`

	// Add dynamic WHERE conditions
	var conditions []string
	if query != "" {
		// Filter by exact title match (case-insensitive)
		conditions = append(conditions, "LOWER(title) = LOWER(?)")
		args = append(args, query)
	}

	// Append WHERE clause if conditions exist
	if len(conditions) > 0 {
		pageQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Order by newest pages first
	pageQuery += " ORDER BY created_at DESC"

	// Execute query
	rows, err := DB.Query(pageQuery, args...)
	if err != nil {
		log.Printf("GetStaticPages query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	// Process each result
	for rows.Next() {
		sp, err := scanStaticPageRow(rows)
		if err != nil {
			return nil, err
		}
		staticPages = append(staticPages, sp)
	}

	return staticPages, nil
}

// scanStaticPageRow scans a database row into a StaticPageRequest struct.
//
// This helper function handles row scanning, datetime formatting, JSON unmarshaling,
// and author enrichment for static page queries.
//
// Parameters:
//   - rows: *sql.Rows - The database rows to scan
//
// Returns:
//   - dtos.StaticPageRequest: Populated static page struct
//   - error: Scan error, unmarshal error, or nil on success
func scanStaticPageRow(rows *sql.Rows) (dtos.StaticPageRequest, error) {
	var sp dtos.StaticPageRequest
	var data sql.NullString
	var userID sql.NullString

	// Scan database row
	if err := rows.Scan(
		&sp.StaticPageID,
		&sp.Title,
		&sp.Description,
		&sp.Path,
		&data,
		&sp.CreatedAt,
		&sp.UpdatedAt,
		&userID,
	); err != nil {
		log.Printf("scanStaticPageRow scan error: %v", err)
		return sp, err
	}

	// Format datetime fields to standard format
	sp.CreatedAt = FormatDateTimeString(sp.CreatedAt)
	sp.UpdatedAt = FormatDateTimeString(sp.UpdatedAt)

	// Unmarshal JSON sections data if present
	if data.Valid && data.String != "" {
		if err := json.Unmarshal([]byte(data.String), &sp.Sections); err != nil {
			log.Printf("scanStaticPageRow unmarshal error: %v", err)
			return sp, err
		}
	}

	// Enrich with author display name
	var err error
	if userID.Valid {
		sp.Author, err = GetUserDisplayName(DB, userID.String)
		if err != nil {
			log.Printf("scanStaticPageRow get user display name error: %v", err)
			return sp, err
		}
	} else {
		// Default for pages without author
		sp.Author = "Unknown"
	}

	return sp, nil
}

// GetStaticPageByID retrieves a single static page by its ID.
//
// This function validates page existence, retrieves full page details,
// unmarshals JSON section data, and enriches with author information.
//
// Parameters:
//   - staticPageID: string - The unique page ID to retrieve
//
// Returns:
//   - *dtos.StaticPageRequest: Page details with:
//   - StaticPageID, Title, Description, Path
//   - Sections: Unmarshaled JSON section data
//   - CreatedAt, UpdatedAt: Formatted datetime strings
//   - Author: Display name of the page creator
//   - error: "static page not found", database error, unmarshal error, or nil on success
func GetStaticPageByID(staticPageID string) (*dtos.StaticPageRequest, error) {
	// Validate page exists
	err := isStaticPageThere(staticPageID)
	if err != nil {
		return nil, err
	}

	var sp dtos.StaticPageRequest
	var (
		data   sql.NullString
		userID sql.NullString
	)

	// Retrieve page details
	err = DB.QueryRow(`
		SELECT static_page_id, title, description, path, data, created_at, updated_at, user_id
		FROM static_pages
		WHERE static_page_id = ?
	`, staticPageID).Scan(&sp.StaticPageID, &sp.Title, &sp.Description, &sp.Path, &data, &sp.CreatedAt, &sp.UpdatedAt, &userID)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON sections into struct
	if err := json.Unmarshal([]byte(data.String), &sp.Sections); err != nil {
		log.Printf("data unmarshal error: %v", err)
		return nil, err
	}

	// Enrich with author display name
	if userID.Valid {
		sp.Author, err = GetUserDisplayName(DB, userID.String)
		if err != nil {
			log.Printf("GetStaticPageByID get user display name error: %v", err)
			return nil, err
		}
	} else {
		// Default for pages without author
		sp.Author = "Unknown"
	}

	// Format datetime fields to standard format
	sp.CreatedAt = FormatDateTimeString(sp.CreatedAt)
	sp.UpdatedAt = FormatDateTimeString(sp.UpdatedAt)

	return &sp, nil
}

// UpdateStaticPage updates an existing static page's content.
//
// This function validates page existence, marshals section data to JSON,
// and updates the page title, description, content data, and path.
//
// Parameters:
//   - staticPageID: string - The unique page ID to update
//   - req: dtos.StaticPageRequest containing updated:
//   - Title: New page title
//   - Description: New page description
//   - Path: New URL path
//   - Sections: Updated page sections (marshaled to JSON)
//
// Returns:
//   - *dtos.StaticPageRequest: The updated page request data
//   - error: "static page not found", database error, or nil on success
func UpdateStaticPage(staticPageID string, req dtos.StaticPageRequest) (*dtos.StaticPageRequest, error) {
	// Validate page exists
	err := isStaticPageThere(staticPageID)
	if err != nil {
		return nil, err
	}

	// Marshal sections to JSON for storage
	data, _ := json.Marshal(req.Sections)

	// Update page details
	query := `
		UPDATE static_pages
		SET title = ?, description = ?, data = ?, path = ?
		WHERE static_page_id = ?
	`
	_, err = DB.Exec(query, req.Title, req.Description, data, req.Path, staticPageID)
	if err != nil {
		return nil, err
	}

	return &req, nil
}

// DeleteStaticPage permanently removes a static page from the system.
//
// This function validates page existence before deletion.
//
// Parameters:
//   - staticPageID: string - The unique page ID to delete
//
// Returns:
//   - error: "static page not found", database error, or nil on success
func DeleteStaticPage(staticPageID string) error {
	// Validate page exists
	err := isStaticPageThere(staticPageID)
	if err != nil {
		return err
	}

	// Delete the page
	query := `
		DELETE FROM static_pages
		WHERE static_page_id = ?
	`
	_, err = DB.Exec(query, staticPageID)
	return err
}

// isStaticPageThere validates that a static page exists in the database.
//
// This helper function is used by update, delete, and get operations
// to ensure the page exists before performing operations.
//
// Parameters:
//   - staticPageID: string - The unique page ID to check
//
// Returns:
//   - error: "static page not found", database error, or nil if page exists
func isStaticPageThere(staticPageID string) error {
	// Check page existence
	exists, err := RecordExists(DB, "static_pages", "static_page_id = ?", staticPageID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("static page not found")
	}
	return nil
}

// FormatDateTimeString formats datetime strings to a standardized format.
//
// This utility function attempts to parse datetime strings from multiple common formats
// and converts them to a consistent "2006-01-02 15:04:05" format for display.
//
// Supported input formats:
//   - RFC3339: "2006-01-02T15:04:05Z07:00"
//   - MySQL datetime: "2006-01-02 15:04:05"
//   - ISO datetime: "2006-01-02T15:04:05"
//   - Date only: "2006-01-02"
//
// Parameters:
//   - dt: string - The datetime string to format
//
// Returns:
//   - string: Formatted datetime as "2006-01-02 15:04:05", or original string if parsing fails,
//     or empty string if input is empty
func FormatDateTimeString(dt string) string {
	// Return empty for empty input
	if dt == "" {
		return ""
	}

	// Define supported datetime formats to try
	layouts := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
		"2006-01-02 15:04:05", // MySQL datetime
		"2006-01-02T15:04:05", // ISO datetime without timezone
		"2006-01-02",          // Date only
	}

	var t time.Time
	var err error

	// Try parsing with each layout until one succeeds
	for _, layout := range layouts {
		t, err = time.Parse(layout, dt)
		if err == nil {
			// Successfully parsed - format to standard output with seconds
			return t.Format("2006-01-02 15:04:05")
		}
	}

	// If no format matched, return original string as fallback
	return dt
}
