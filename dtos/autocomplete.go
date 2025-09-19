package dtos

// AutoCompleteResult represents a single autocomplete suggestion
type AutoCompleteResult struct {
	Type        string `json:"type,omitempty"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	ImageURL    string `json:"image_url,omitempty"`
	Link        string `json:"link"`
}

// AutoCompleteResponse represents the autocomplete API response
type AutoCompleteResponse struct {
	Suggestions []AutoCompleteResult `json:"suggestions"`
	Total       int                  `json:"total"`
}
