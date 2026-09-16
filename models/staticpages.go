package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"errors"
	"log"
	"strings"

	"github.com/teris-io/shortid"
)

func CreateStaticPage(req dtos.StaticPageRequest, userID string) error {
	// Validate title and path uniqueness (case-insensitive)
	err := isStaticPageThereByTitleOrPath(req.Title, req.Path)
	if err != nil {
		return err
	}

	// Generate unique page ID
	staticPageID, _ := shortid.Generate()

	// Insert new static page
	query := `
		INSERT INTO static_pages (static_page_id, title, content, path, user_id)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(query, staticPageID, req.Title, req.Content, req.Path, userID)
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
		args        []any
	)

	// Build base query
	pageQuery := `
		SELECT static_page_id, title, path, content, created_at, updated_at, user_id
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
	var userID sql.NullString

	// Scan database row
	if err := rows.Scan(
		&sp.StaticPageID,
		&sp.Title,
		&sp.Path,
		&sp.Content,
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
		userID sql.NullString
	)

	// Retrieve page details
	err = DB.QueryRow(`
		SELECT static_page_id, title, path, content, created_at, updated_at, user_id
		FROM static_pages
		WHERE static_page_id = ?
	`, staticPageID).Scan(&sp.StaticPageID, &sp.Title, &sp.Path, &sp.Content, &sp.CreatedAt, &sp.UpdatedAt, &userID)
	if err != nil {
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

	// Update page details
	query := `
		UPDATE static_pages
		SET title = ?, content = ?, path = ?
		WHERE static_page_id = ?
	`
	_, err = DB.Exec(query, req.Title, req.Content, req.Path, staticPageID)
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
