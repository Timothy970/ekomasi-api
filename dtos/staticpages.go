package dtos

// {
//   "title": "Terms & Condition",
//   "slug": "terms-and-condition",
//   "content": "<p>Your full HTML or rich-text content here...</p>",
//   "page_type": "general",
//   "status": "draft"
// }

type StaticPageRequest struct {
	Title    string `json:"title" validate:"required"`
	Slug     string `json:"slug" validate:"required"`
	Content  string `json:"content" validate:"required"`
	PageType string `json:"page_type" validate:"required"`
	Status   string `json:"status" validate:"required"`
}
