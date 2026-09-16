package models

import (
	"database/sql"
	"ekomasi_backend/dtos"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

func CreateBlog(blog dtos.BlogRequest, authorID string) error {
	// Generate unique blog ID
	blogID, _ := shortid.Generate()

	var publishedAt *string
	isPublished := true

	// Default status to "published" if not provided
	if blog.Status == nil {
		blog.Status = new(string)
		*blog.Status = "published"
	}
	// Published: set published_at to current timestamp
	now := time.Now().Format("2006-01-02 15:04:05")
	publishedAt = &now

	// Handle draft vs published status
	if strings.ToLower(*blog.Status) == "draft" {
		// Draft: unpublished with no published_at date
		isPublished = false
	}

	// Marshal content sections to JSON
	contentData, err := json.Marshal(blog.Sections)
	if err != nil {
		fmt.Println("Error converting to JSON:", err)
		return err
	}
	name, _ := getUserNames(blog.AuthorID)

	blog.Author = map[string]any{
		"avatar": "",
		"name":   name,
	}

	// Marshal author information to JSON
	authorData, err := json.Marshal(blog.Author)
	if err != nil {
		fmt.Println("Error converting author to JSON:", err)
		return err
	}

	// Marshal tags to JSON
	tagData, err := json.Marshal(blog.Tags)
	if err != nil {
		fmt.Println("Error converting tags to JSON:", err)
		return err
	}

	// Insert blog record with JSON fields
	query := `
		INSERT INTO blogs (blog_id, title, content, author_id, published_at, is_published, author, tags, description, read_time, status, banner_image_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = DB.Exec(query, blogID, blog.Title, contentData, authorID, publishedAt, isPublished, authorData, tagData, blog.Description, blog.ReadTimeMinutes, blog.Status, blog.BannerImageUrl)
	return err
}

// GetBlogByID retrieves a single blog post by its blog_id.
//
// This function fetches complete blog data including JSON fields (content sections,
// author info, tags) and deserializes them into structured objects.
//
// Parameters:
//   - blogID: The blog_id to retrieve
//
// Returns:
//   - *dtos.BlogRequest: Pointer to blog with deserialized JSON fields
//   - error: "blog not found" if blog doesn't exist,
//     JSON unmarshal error if JSON invalid,
//     or database error
//
// JSON Fields:
//   - Sections: Array of content sections
//   - Author: Author information object
//   - Tags: Array of tag strings
func GetBlogByID(blogID string) (*dtos.BlogRequest, error) {
	exists, err := RecordExists(DB, "blogs", fetchblog, blogID)
	if err != nil {
		return nil, fmt.Errorf("failed to check blog existence: %w", err)
	}
	if !exists {
		return nil, errors.New(noblog)
	}

	query := `
		SELECT blog_id, title, content, author_id, published_at, is_published, 
		       author, tags, description, read_time, status, created_at, updated_at, banner_image_url
		FROM blogs
		WHERE blog_id = ?
	`

	var (
		blog        dtos.BlogRequest
		contentJSON sql.NullString
		authorJSON  sql.NullString
		tagsJSON    sql.NullString
		publishedAt sql.NullTime
		readTime    sql.NullInt64
	)

	row := DB.QueryRow(query, blogID)
	if err := row.Scan(
		&blog.BlogID, &blog.Title, &contentJSON, &blog.AuthorID,
		&publishedAt, &blog.IsPublished, &authorJSON, &tagsJSON,
		&blog.Description, &readTime, &blog.Status,
		&blog.CreatedAt, &blog.UpdatedAt, &blog.BannerImageUrl,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New(noblog)
		}
		return nil, fmt.Errorf("failed to scan blog: %w", err)
	}

	unmarshalJSONField := func(data sql.NullString, target any, field string) error {
		if data.Valid {
			if err := json.Unmarshal([]byte(data.String), target); err != nil {
				return fmt.Errorf("invalid %s JSON: %w", field, err)
			}
		}
		return nil
	}

	if err := unmarshalJSONField(contentJSON, &blog.Sections, "content"); err != nil {
		return nil, err
	}
	if err := unmarshalJSONField(authorJSON, &blog.Author, "author"); err != nil {
		return nil, err
	}
	if err := unmarshalJSONField(tagsJSON, &blog.Tags, "tags"); err != nil {
		return nil, err
	}

	if publishedAt.Valid {
		blog.PublishedAt = publishedAt.Time
	}
	if readTime.Valid {
		blog.ReadTimeMinutes = int(readTime.Int64)
	}

	return &blog, nil
}

// UpdateBlog updates an existing blog post with full field replacement.
//
// This function validates blog existence and updates all fields including JSON
// content. It handles status transitions and published_at accordingly.
//
// Parameters:
//   - blog: dtos.BlogRequest with fields to update:
//   - Title: Updated title
//   - Sections: Updated content sections (marshaled to JSON)
//   - Author: Updated author info (marshaled to JSON)
//   - Tags: Updated tags (marshaled to JSON)
//   - Description: Updated summary
//   - ReadTimeMinutes: Updated read time
//   - Status: "draft" or "published" (defaults to "published" if nil)
//   - BannerImageUrl: Updated banner image
//   - blogID: The blog_id to update
//
// Returns:
//   - error: "blog not found" if blog doesn't exist,
//     JSON marshaling error,
//     or database error
//
// Status Transitions:
//   - draft → published: Sets published_at to NOW(), is_published=true
//   - published → draft: Sets published_at=NULL, is_published=false
func UpdateBlog(blog dtos.BlogRequest, blogID string) error {
	// Verify blog exists
	exists, err := RecordExists(DB, "blogs", fetchblog, blogID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noblog)
	}

	var publishedAt *string
	isPublished := true

	// Default status to "published" if not provided
	if blog.Status == nil {
		blog.Status = new(string)
		*blog.Status = "published"
	}

	// Handle draft vs published status
	if strings.ToLower(*blog.Status) == "draft" {
		// Draft: unpublished with no published_at date
		isPublished = false
		publishedAt = nil // ← represents NULL in DB
	} else {
		// Published: set published_at to current timestamp
		now := time.Now().Format("2006-01-02 15:04:05")
		publishedAt = &now

	}

	// Marshal content sections to JSON
	contentData, err := json.Marshal(blog.Sections)
	if err != nil {
		fmt.Println("Error converting to JSON:", err)
		return err
	}
	name, _ := getUserNames(blog.AuthorID)
	blog.Author = map[string]any{
		"avatar": "",
		"name":   name,
	}
	// Marshal author information to JSON
	authorData, err := json.Marshal(blog.Author)
	if err != nil {
		fmt.Println("Error converting author to JSON:", err)
		return err
	}

	// Marshal tags to JSON
	tagData, err := json.Marshal(blog.Tags)
	if err != nil {
		fmt.Println("Error converting tags to JSON:", err)
		return err
	}

	// Update all blog fields
	query := `
		UPDATE blogs SET title = ?, content = ?, published_at = ?, is_published = ?, author = ?, tags = ?, description = ?, read_time = ?, status = ?, banner_image_url = ?
		WHERE blog_id = ?`

	_, err = DB.Exec(query, blog.Title, contentData, publishedAt, isPublished, authorData, tagData, blog.Description, blog.ReadTimeMinutes, blog.Status, blog.BannerImageUrl, blogID)
	return err
}

// DeleteBlog permanently removes a blog post and returns error if not found.
//
// Parameters:
//   - blogID: The blog_id to delete
//
// Returns:
//   - error: "blog not found" if blog doesn't exist or database error
func DeleteBlog(blogID string) error {
	// Verify blog exists
	exists, err := RecordExists(DB, "blogs", fetchblog, blogID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(noblog)
	}

	// Delete blog record
	query := `DELETE FROM blogs WHERE blog_id = ?`
	_, err = DB.Exec(query, blogID)
	return err
}

// ListBlogs retrieves paginated list of blogs with optional status filtering.
//
// This function fetches blogs with pagination metadata and supports filtering
// by blog status (draft/published). Results are ordered by publish date descending.
//
// Parameters:
//   - page: Page number (1-indexed)
//   - limit: Items per page
//   - status: Filter by status ("draft", "published", or "" for all)
//
// Returns:
//   - []dtos.BlogRequest: Array of blogs with deserialized JSON fields
//   - *dtos.PaginationMeta: Pagination info (total, pages, current page)
//   - error: Database error or JSON unmarshal error
func ListBlogs(page, limit int, title string, isAdmin bool) ([]dtos.BlogRequest, *dtos.PaginationMeta, error) {
	// Calculate offset for pagination
	offset := (page - 1) * limit

	// Get total count for pagination metadata
	totalItems, err := countBlogs(title, isAdmin)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count blogs: %w", err)
	}

	// Fetch blog records
	rows, err := fetchBlogs(title, limit, offset, isAdmin)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query blogs: %w", err)
	}
	defer rows.Close()

	// Deserialize blog rows
	blogs, err := scanBlogs(rows)
	if err != nil {
		return nil, nil, err
	}

	// Build pagination metadata
	meta := buildPagination(limit, offset, totalItems)
	return blogs, meta, nil
}

// countBlogs returns total blog count with optional status filtering.
//
// This is an internal helper function used for pagination calculations.
//
// Parameters:
//   - status: Filter by status ("draft", "published", or "" for all)
//
// Returns:
//   - int: Total number of blogs matching filter
//   - error: Database error if query fails
