package dtos

// {
//   "title": "Terms & Condition",
//   "slug": "terms-and-condition",
//   "content": "<p>Your full HTML or rich-text content here...</p>",
//   "page_type": "general",
//   "status": "draft"
// }

type StaticPageRequest struct {
	Author       string `json:"author"`
	StaticPageID string `json:"static_page_id"`
	Title        string `json:"title" validate:"required"`
	Path         string `json:"path" validate:"required"`
	Content      string `json:"content" validate:"required"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type StaticPageContent struct {
	Header map[string]any `json:"header"`
	Body   map[string]any `json:"body"`
	Footer map[string]any `json:"footer"`
}
