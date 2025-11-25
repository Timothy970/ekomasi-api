package models

import (
	"adenzo_backend/dtos"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/teris-io/shortid"
)

func CreateStaticPage(req dtos.StaticPageRequest) error {
	staticPageID, _ := shortid.Generate()
	data, _ := json.Marshal(req.Sections)
	query := `
		INSERT INTO static_pages (static_page_id, title, description, data, path)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := DB.Exec(query, staticPageID, req.Title, req.Description, data, req.Path)
	return err
}

func GetStaticPages(query string) ([]dtos.StaticPageRequest, error) {
	var (
		staticPages []dtos.StaticPageRequest
		args        []interface{}
	)

	// Base query
	pageQuery := `
		SELECT static_page_id, title, description, path, data, created_at, updated_at
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

		if err := rows.Scan(
			&sp.StaticPageID,
			&sp.Title,
			&sp.Description,
			&sp.Path,
			&data,
			&sp.CreatedAt,
			&sp.UpdatedAt,
		); err != nil {
			log.Printf("GetStaticPages scan error: %v", err)
			return nil, err
		}

		// Only unmarshal if data is valid JSON
		if data.Valid && data.String != "" {
			if err := json.Unmarshal([]byte(data.String), &sp.Sections); err != nil {
				log.Printf("GetStaticPages unmarshal error: %v", err)
				return nil, err
			}
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
		data sql.NullString
	)
	err = DB.QueryRow(`
		SELECT static_page_id, title, description, path, data, created_at, updated_at
		FROM static_pages
		WHERE static_page_id = ?
	`, staticPageID).Scan(&sp.StaticPageID, &sp.Title, &sp.Description, &sp.Path, &data, &sp.CreatedAt, &sp.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON data into the content struct
	if err := json.Unmarshal([]byte(data.String), &sp.Sections); err != nil {
		log.Printf("data unmarshal error: %v", err)
		return nil, err
	}
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
