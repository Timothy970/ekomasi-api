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
	headerData, _ := json.Marshal(req.Content.Header)
	bodyData, _ := json.Marshal(req.Content.Body)
	footerData, _ := json.Marshal(req.Content.Footer)
	query := `
		INSERT INTO static_pages (static_page_id, title, slug, body_data, header_data, footer_data, page_type, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := DB.Exec(query, staticPageID, req.Title, req.Slug, bodyData, headerData, footerData, req.PageType, req.Status)
	return err
}

func GetStaticPages(slug, pageType, status string, page, limit int) ([]dtos.StaticPageRequest, *dtos.PaginationMeta, error) {
	var count int
	baseQuery := `SELECT COUNT(*) FROM static_pages`
	conditions := []string{}
	args := []interface{}{}

	// Build dynamic conditions
	if slug != "" {
		conditions = append(conditions, "LOWER(slug) = LOWER(?)")
		args = append(args, slug)
	}
	if pageType != "" {
		conditions = append(conditions, "LOWER(page_type) = LOWER(?)")
		args = append(args, pageType)
	}
	if status != "" {
		conditions = append(conditions, "LOWER(status) = LOWER(?)")
		args = append(args, status)
	}

	// Add WHERE clause if conditions exist
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Execute safely with only the needed arguments
	if err := DB.QueryRow(baseQuery, args...).Scan(&count); err != nil {
		log.Printf("count query exec error: %v", err)
		return nil, nil, err
	}
	// Fetch the actual static pages
	var staticPages []dtos.StaticPageRequest

	pageQuery := `
	SELECT static_page_id, title, slug, header_data, body_data, footer_data, page_type, status
	FROM static_pages
`
	pageConditions := []string{}
	pageArgs := []interface{}{}

	// Build dynamic conditions
	if slug != "" {
		pageConditions = append(pageConditions, "LOWER(slug) = LOWER(?)")
		pageArgs = append(pageArgs, slug)
	}
	if pageType != "" {
		pageConditions = append(pageConditions, "LOWER(page_type) = LOWER(?)")
		pageArgs = append(pageArgs, pageType)
	}
	if status != "" {
		pageConditions = append(pageConditions, "LOWER(status) = LOWER(?)")
		pageArgs = append(pageArgs, status)
	}

	// Append WHERE clause dynamically
	if len(pageConditions) > 0 {
		pageQuery += " WHERE " + strings.Join(pageConditions, " AND ")
	}

	// Add pagination
	pageQuery += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, (page-1)*limit)

	// Execute query safely
	rows, err := DB.Query(pageQuery, args...)
	if err != nil {
		log.Printf("page query exec error: %v", err)
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sp dtos.StaticPageRequest
		var (
			headerData sql.NullString
			bodyData   sql.NullString
			footerData sql.NullString
		)

		if err := rows.Scan(&sp.StaticPageID, &sp.Title, &sp.Slug, &headerData, &bodyData, &footerData, &sp.PageType, &sp.Status); err != nil {
			log.Printf("row scan error***%s", err)
			return nil, nil, err
		}

		// Unmarshal JSON data into the content struct
		if err := json.Unmarshal([]byte(headerData.String), &sp.Content.Header); err != nil {
			log.Printf("header unmarshal error: %v", err)
			return nil, nil, err
		}
		if err := json.Unmarshal([]byte(bodyData.String), &sp.Content.Body); err != nil {
			log.Printf("body unmarshal error: %v", err)
			return nil, nil, err
		}
		if err := json.Unmarshal([]byte(footerData.String), &sp.Content.Footer); err != nil {
			log.Printf("footer unmarshal error: %v", err)
			return nil, nil, err
		}

		staticPages = append(staticPages, sp)
	}

	// Create pagination metadata
	pagination := calculatePagination(page, limit, int64(count))

	return staticPages, &pagination, nil
}
func GetStaticPageByID(staticPageID string) (*dtos.StaticPageRequest, error) {
	err := isStaticPageThere(staticPageID)
	if err != nil {
		return nil, err
	}
	var sp dtos.StaticPageRequest
	var (
		headerData sql.NullString
		bodyData   sql.NullString
		footerData sql.NullString
	)
	err = DB.QueryRow(`
		SELECT static_page_id, title, slug, header_data, body_data, footer_data, page_type, status
		FROM static_pages
		WHERE static_page_id = ?
	`, staticPageID).Scan(&sp.StaticPageID, &sp.Title, &sp.Slug, &headerData, &bodyData, &footerData, &sp.PageType, &sp.Status)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON data into the content struct
	if err := json.Unmarshal([]byte(headerData.String), &sp.Content.Header); err != nil {
		log.Printf("header unmarshal error: %v", err)
		return nil, err
	}
	if err := json.Unmarshal([]byte(bodyData.String), &sp.Content.Body); err != nil {
		log.Printf("body unmarshal error: %v", err)
		return nil, err
	}
	if err := json.Unmarshal([]byte(footerData.String), &sp.Content.Footer); err != nil {
		log.Printf("footer unmarshal error: %v", err)
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
	query := `
		UPDATE static_pages
		SET title = ?, slug = ?, content = ?, page_type = ?, status = ?
		WHERE static_page_id = ?
	`
	_, err = DB.Exec(query, req.Title, req.Slug, req.Content, req.PageType, req.Status, staticPageID)
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
