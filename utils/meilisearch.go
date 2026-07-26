package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type MeiliProductDocument struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenant_id"`
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Price       float64  `json:"price"`
	ImageURL    string   `json:"image_url"`
	InStock     bool     `json:"in_stock"`
	Tags        []string `json:"tags"`
}

// IndexProductInMeilisearch pushes product updates to Meilisearch with tenant_id isolation
func IndexProductInMeilisearch(doc MeiliProductDocument) error {
	host := os.Getenv("MEILISEARCH_HOST")
	apiKey := os.Getenv("MEILISEARCH_KEY")

	if host == "" {
		host = "http://127.0.0.1:7700"
	}

	url := fmt.Sprintf("%s/indexes/products/documents", host)
	payload, err := json.Marshal([]MeiliProductDocument{doc})
	if err != nil {
		return fmt.Errorf("failed to marshal meilisearch doc: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create meilisearch request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("meilisearch connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("meilisearch returned status %d", resp.StatusCode)
	}
	return nil
}
