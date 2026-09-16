package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"errors"
	"fmt"
)

func countBlogs(title string, isAdmin bool) (int, error) {
	// Build count query with optional title filter
	query := "SELECT COUNT(*) FROM blogs"
	var args []any

	// Add title filter if provided
	if title != "" {
		query += " WHERE title LIKE ?"
		args = append(args, "%"+title+"%")
	}

	//if not admin, only count published blogs
	if !isAdmin {
		if title != "" {
			query += " AND status = 'published'"
		} else {
			query += " WHERE status = 'published'"
		}
	}

	// Execute count query
	var total int
	if err := DB.QueryRow(query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// fetchBlogs retrieves blog rows with pagination and optional status filtering.
//
// This is an internal helper function that returns sql.Rows for scanning.
//
// Parameters:
//   - status: Filter by status ("draft", "published", or "" for all)
//   - limit: Maximum number of rows to return
//   - offset: Number of rows to skip
//
// Returns:
//   - *sql.Rows: Result set with blog records (caller must close)
//   - error: Database error if query fails
func fetchBlogs(title string, limit, offset int, isAdmin bool) (*sql.Rows, error) {
	// Query all blog fields with JSON columns
	query := `
		SELECT blog_id, title, content, author_id, published_at, is_published, 
		       author, tags, description, read_time, status, created_at, updated_at, banner_image_url
		FROM blogs
	`
	var args []any

	// Add title filter if provided
	if title != "" {
		query += " WHERE title LIKE ?"
		args = append(args, "%"+title+"%")
	}

	//if not admin, only fetch published blogs
	if !isAdmin {
		if title != "" {
			query += " AND status = 'published'"
		} else {
			query += " WHERE status = 'published'"
		}
	}

	// Add ordering and pagination
	query += " ORDER BY published_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	return DB.Query(query, args...)
}

// scanBlogs deserializes result rows into blog objects with JSON unmarshaling.
//
// This is an internal helper function that handles row scanning and JSON
// deserialization for blog list queries.
//
// Parameters:
//   - rows: Result set from fetchBlogs (or similar query)
//
// Returns:
//   - []dtos.BlogRequest: Array of blogs with deserialized JSON fields
//   - error: Scan error or JSON unmarshal error
//
// JSON Fields Deserialized:
//   - content: Sections array
//   - author: Author object
//   - tags: Tags array
func scanBlogs(rows *sql.Rows) ([]dtos.BlogRequest, error) {
	var blogs []dtos.BlogRequest

	// Iterate through result rows
	for rows.Next() {
		var (
			blog        dtos.BlogRequest
			contentJSON sql.NullString
			authorJSON  sql.NullString
			tagsJSON    sql.NullString
			publishedAt sql.NullTime
			readTime    sql.NullInt64
		)

		// Scan row into blog struct and JSON fields
		if err := rows.Scan(
			&blog.BlogID, &blog.Title, &contentJSON, &blog.AuthorID,
			&publishedAt, &blog.IsPublished, &authorJSON, &tagsJSON,
			&blog.Description, &readTime, &blog.Status,
			&blog.CreatedAt, &blog.UpdatedAt, &blog.BannerImageUrl,
		); err != nil {
			return nil, fmt.Errorf("failed to scan blog: %w", err)
		}

		// Parse JSON fields and nullable fields
		if err := parseBlogFields(&blog, contentJSON, authorJSON, tagsJSON, publishedAt, readTime); err != nil {
			return nil, err
		}

		blogs = append(blogs, blog)
	}
	return blogs, nil
}

// parseBlogFields deserializes JSON fields and nullable fields into blog struct.
//
// This is an internal helper function that handles JSON unmarshaling and
// nullable field conversion for blog objects.
//
// Parameters:
//   - blog: Pointer to blog struct to populate
//   - contentJSON: JSON content sections (nullable)
//   - authorJSON: JSON author info (nullable)
//   - tagsJSON: JSON tags array (nullable)
//   - publishedAt: Published timestamp (nullable)
//   - readTime: Reading time in minutes (nullable)
//
// Returns:
//   - error: JSON unmarshal error if any JSON field is invalid
//
// JSON Handling:
//   - Valid JSON is unmarshaled into corresponding struct fields
//   - NULL values are handled gracefully (fields remain zero-valued)
func parseBlogFields(
	blog *dtos.BlogRequest,
	contentJSON, authorJSON, tagsJSON sql.NullString,
	publishedAt sql.NullTime,
	readTime sql.NullInt64,
) error {
	// Generic JSON unmarshal helper
	unmarshal := func(data sql.NullString, target any, field string) error {
		if data.Valid {
			if err := json.Unmarshal([]byte(data.String), target); err != nil {
				return fmt.Errorf("invalid %s JSON: %w", field, err)
			}
		}
		return nil
	}

	// Unmarshal JSON fields
	if err := unmarshal(contentJSON, &blog.Sections, "content"); err != nil {
		return err
	}
	if err := unmarshal(authorJSON, &blog.Author, "author"); err != nil {
		return err
	}
	if err := unmarshal(tagsJSON, &blog.Tags, "tags"); err != nil {
		return err
	}

	// Convert nullable timestamp and int fields
	if publishedAt.Valid {
		blog.PublishedAt = publishedAt.Time
	}
	if readTime.Valid {
		blog.ReadTimeMinutes = int(readTime.Int64)
	}
	return nil
}

// DeleteBanner permanently removes a banner and returns error if not found.
//
// Parameters:
//   - bannerID: The banner ID to delete
//
// Returns:
//   - error: "banner not found" if banner doesn't exist or database error
func DeleteBanner(bannerID string) error {
	// Verify banner exists
	exists, err := RecordExists(DB, "banners", whereID, bannerID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("banner not found")
	}

	// Delete banner record
	query := `DELETE FROM banners WHERE id = ?`
	_, err = DB.Exec(query, bannerID)
	return err
}

// CreateMenuLink creates a new menu link with optional parent (for hierarchy).
//
// This function creates navigation menu links with optional parent-child relationships.
// Menu titles must be unique across all menu links.
//
// Parameters:
//   - req: dtos.MenuLinkRequest containing:
//   - Title: Menu link title (must be unique)
//   - URL: Link destination URL
//   - DisplayOrder: Position ordering
//   - ParentID: Optional parent menu ID for nested menus (nil for top-level)
//
// Returns:
//   - error: "menu title already exists" if title is duplicate,
//     or database error
//
// Parent Handling:
//   - ParentID=nil: Creates top-level menu item
//   - ParentID set: Creates submenu under specified parent
func CreateMenuLink(req dtos.MenuLinkRequest) error {
	// Validate menu title uniqueness
	exists, err := RecordExists(DB, "menu_links", "title = ?", req.Title)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("menu title already exists")
	}

	// Handle parent-child relationship
	if req.ParentID == nil {
		// Insert top-level menu (no parent)
		query := `INSERT INTO menu_links (title, url, display_order) VALUES (?, ?, ?)`
		_, err := DB.Exec(query, req.Title, req.URL, req.DisplayOrder)
		if err != nil {
			return err
		}

		// // Get inserted ID
		// id, err := result.LastInsertId()
		// if err != nil {
		// 	return err
		// }

		// // Update parent_id to self
		// _, err = DB.Exec(`UPDATE menu_links SET parent_id = ? WHERE id = ?`, id, id)
		// if err != nil {
		// 	return err
		// }

	} else {
		// Insert submenu with parent reference
		query := `INSERT INTO menu_links (title, url, display_order, parent_id) VALUES (?, ?, ?, ?)`
		_, err = DB.Exec(query, req.Title, req.URL, req.DisplayOrder, req.ParentID)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateMenuLink updates an existing menu link's properties.
//
// Parameters:
//   - menu: dtos.MenuLinkRequest with updated values:
//   - Title: Updated menu title
//   - URL: Updated link destination
//   - DisplayOrder: Updated position
//   - menuLinkID: The menu link ID to update
//
// Returns:
//   - error: "menu link not found" if menu doesn't exist or database error
func UpdateMenuLink(menu dtos.MenuLinkRequest, menuLinkID int) error {
	// Verify menu link exists
	exists, err := RecordExists(DB, "menu_links", whereID, menuLinkID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("menu link not found")
	}

	// Update menu link properties
	query := `UPDATE menu_links SET title=?, url=?, display_order=? WHERE id=?`
	_, err = DB.Exec(query, menu.Title, menu.URL, menu.DisplayOrder, menuLinkID)
	if err != nil {
		return err
	}
	return nil
}

// DeleteMenuLink permanently removes a menu link.
//
// Parameters:
//   - menuLinkID: The menu link ID to delete
//
// Returns:
//   - error: "menu link not found" if menu doesn't exist or database error
