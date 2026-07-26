package utils

import (
	"encoding/json"
	"fmt"
)

type ProductSchemaLD struct {
	Context     string  `json:"@context"`
	Type        string  `json:"@type"`
	Name        string  `json:"name"`
	Image       string  `json:"image,omitempty"`
	Description string  `json:"description,omitempty"`
	SKU         string  `json:"sku,omitempty"`
	Offers      OfferLD `json:"offers"`
}

type OfferLD struct {
	Type          string  `json:"@type"`
	PriceCurrency string  `json:"priceCurrency"`
	Price         float64 `json:"price"`
	Availability  string  `json:"availability"`
	URL           string  `json:"url,omitempty"`
}

// GenerateProductJSONLD constructs a valid Schema.org Product JSON-LD string
func GenerateProductJSONLD(name, image, description, sku, currency string, price float64, inStock bool, productURL string) (string, error) {
	avail := "https://schema.org/InStock"
	if !inStock {
		avail = "https://schema.org/OutOfStock"
	}

	if currency == "" {
		currency = "KES"
	}

	ld := ProductSchemaLD{
		Context:     "https://schema.org",
		Type:        "Product",
		Name:        name,
		Image:       image,
		Description: description,
		SKU:         sku,
		Offers: OfferLD{
			Type:          "Offer",
			PriceCurrency: currency,
			Price:         price,
			Availability:  avail,
			URL:           productURL,
		},
	}

	bytes, err := json.MarshalIndent(ld, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON-LD: %w", err)
	}

	return string(bytes), nil
}
