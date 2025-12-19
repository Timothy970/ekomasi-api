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

func CreateStaticPage(req dtos.StaticPageRequest, userID string) error {
	//check if title or path already exists
	err := isStaticPageThereByTitleOrPath(req.Title, req.Path)
	staticPageID, _ := shortid.Generate()
	data, _ := json.Marshal(req.Sections)
	query := `
		INSERT INTO static_pages (static_page_id, title, description, data, path, user_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = DB.Exec(query, staticPageID, req.Title, req.Description, data, req.Path, userID)
	return err
}

func isStaticPageThereByTitleOrPath(title, path string) error {
	//first check if title exists for uniqueness
	exists, err := RecordExists("static_pages", "LOWER(title) = LOWER(?) = LOWER(?)", title)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("static page with this title already exists")
	}
	//then check if path exists for uniqueness
	exists, err = RecordExists("static_pages", "LOWER(path) = LOWER(?)", path)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("static page with this path already exists")
	}
	return nil
}

func GetStaticPages(query string) ([]dtos.StaticPageRequest, error) {
	var (
		staticPages []dtos.StaticPageRequest
		args        []interface{}
	)

	// Base query
	pageQuery := `
		SELECT static_page_id, title, description, path, data, created_at, updated_at, user_id
		FROM static_pages
	`

	// Build dynamic conditions
	var conditions []string
	if query != "" {
		conditions = append(conditions, "LOWER(title) = LOWER(?)")
		args = append(args, query)
	}

	// Append WHERE clause dynamically
	if len(conditions) > 0 {
		pageQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	pageQuery += " ORDER BY created_at DESC"

	// Execute the query
	rows, err := DB.Query(pageQuery, args...)
	if err != nil {
		log.Printf("GetStaticPages query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	// Iterate results
	for rows.Next() {
		var sp dtos.StaticPageRequest
		var data sql.NullString
		var userID sql.NullString

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
			log.Printf("GetStaticPages scan error: %v", err)
			return nil, err
		}

		// Format created_at and updated_at
		sp.CreatedAt = FormatDateTimeString(sp.CreatedAt)
		sp.UpdatedAt = FormatDateTimeString(sp.UpdatedAt)

		// Only unmarshal if data is valid JSON
		if data.Valid && data.String != "" {
			if err := json.Unmarshal([]byte(data.String), &sp.Sections); err != nil {
				log.Printf("GetStaticPages unmarshal error: %v", err)
				return nil, err
			}
		}
		if userID.Valid {
			sp.Author, err = GetUserDisplayName(userID.String)
			if err != nil {
				log.Printf("GetStaticPages get user display name error: %v", err)
				return nil, err
			}
		} else {
			sp.Author = "Unknown"
		}

		staticPages = append(staticPages, sp)
	}

	return staticPages, nil
}

func GetStaticPageByID(staticPageID string) (*dtos.StaticPageRequest, error) {
	err := isStaticPageThere(staticPageID)
	if err != nil {
		return nil, err
	}
	var sp dtos.StaticPageRequest
	var (
		data   sql.NullString
		userID sql.NullString
	)
	err = DB.QueryRow(`
		SELECT static_page_id, title, description, path, data, created_at, updated_at, user_id
		FROM static_pages
		WHERE static_page_id = ?
	`, staticPageID).Scan(&sp.StaticPageID, &sp.Title, &sp.Description, &sp.Path, &data, &sp.CreatedAt, &sp.UpdatedAt, &userID)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON data into the content struct
	if err := json.Unmarshal([]byte(data.String), &sp.Sections); err != nil {
		log.Printf("data unmarshal error: %v", err)
		return nil, err
	}
	if userID.Valid {
		sp.Author, err = GetUserDisplayName(userID.String)
		if err != nil {
			log.Printf("GetStaticPageByID get user display name error: %v", err)
			return nil, err
		}
	} else {
		sp.Author = "Unknown"
	}

	// Format created_at and updated_at
	sp.CreatedAt = FormatDateTimeString(sp.CreatedAt)
	sp.UpdatedAt = FormatDateTimeString(sp.UpdatedAt)
	return &sp, nil
}
func UpdateStaticPage(staticPageID string, req dtos.StaticPageRequest) (*dtos.StaticPageRequest, error) {
	// Check if static page exists
	err := isStaticPageThere(staticPageID)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(req.Sections)
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
func DeleteStaticPage(staticPageID string) error {
	// Check if static page exists
	err := isStaticPageThere(staticPageID)
	if err != nil {
		return err
	}
	query := `
		DELETE FROM static_pages
		WHERE static_page_id = ?
	`
	_, err = DB.Exec(query, staticPageID)
	return err
}
func isStaticPageThere(staticPageID string) error {
	exists, err := RecordExists("static_pages", "static_page_id = ?", staticPageID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("static page not found")
	}
	return nil
}

// formatDateTimeString formats a datetime string into "2006-01-02 15:04"
func FormatDateTimeString(dt string) string {
	if dt == "" {
		return ""
	}

	// Try parsing multiple common datetime formats
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02", // date only
	}

	var t time.Time
	var err error

	for _, layout := range layouts {
		t, err = time.Parse(layout, dt)
		if err == nil {
			// Successful parse - return with seconds
			return t.Format("2006-01-02 15:04:05")
		}
	}

	// If nothing works, return original
	return dt
}
