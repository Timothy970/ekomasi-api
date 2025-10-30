package dtos

// {
//   "title": "Terms & Condition",
//   "slug": "terms-and-condition",
//   "content": "<p>Your full HTML or rich-text content here...</p>",
//   "page_type": "general",
//   "status": "draft"
// }

type StaticPageRequest struct {
	StaticPageID string            `json:"static_page_id"`
	Title        string            `json:"title" validate:"required"`
	Slug         string            `json:"slug" validate:"required"`
	Content      StaticPageContent `json:"content" dive:"required"`
	PageType     string            `json:"page_type" validate:"required"`
	Status       string            `json:"status" validate:"required"`
}
type StaticPageContent struct {
	Header map[string]any `json:"header"`
	Body   map[string]any `json:"body"`
	Footer map[string]any `json:"footer"`
}
