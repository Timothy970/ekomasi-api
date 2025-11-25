package dtos

// {
//   "title": "Terms & Condition",
//   "slug": "terms-and-condition",
//   "content": "<p>Your full HTML or rich-text content here...</p>",
//   "page_type": "general",
//   "status": "draft"
// }

type StaticPageRequest struct {
	StaticPageID string  `json:"static_page_id"`
	Title        string  `json:"title" validate:"required"`
	Path         string  `json:"path" validate:"required"`
	Description  *string `json:"description" validate:"required"`
	Sections     []struct {
		Position   int         `json:"position" validate:"required"`
		Banner     *BlogImage  `json:"banner"`
		Title      string      `json:"title"`
		Paragraphs []Paragraph `json:"paragraphs" dive:"required"`
		Images     []BlogImage `json:"images"`
	} `json:"sections"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type StaticPageContent struct {
	Header map[string]any `json:"header"`
	Body   map[string]any `json:"body"`
	Footer map[string]any `json:"footer"`
}
