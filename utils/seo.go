package utils

import (
	"encoding/json"
	"fmt"
)

type ProductJSONLDParams struct {
	Name        string
	Image       string
	Description string
	SKU         string
	Currency    string
	Price       float64
	InStock     bool
	ProductURL  string
}

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
func GenerateProductJSONLD(params ProductJSONLDParams) (string, error) {
	avail := "https://schema.org/InStock"
	if !params.InStock {
		avail = "https://schema.org/OutOfStock"
	}

	currency := params.Currency
	if currency == "" {
		currency = "KES"
	}

	ld := ProductSchemaLD{
		Context:     "https://schema.org",
		Type:        "Product",
		Name:        params.Name,
		Image:       params.Image,
		Description: params.Description,
		SKU:         params.SKU,
		Offers: OfferLD{
			Type:          "Offer",
			PriceCurrency: currency,
			Price:         params.Price,
			Availability:  avail,
			URL:           params.ProductURL,
		},
	}

	bytes, err := json.MarshalIndent(ld, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON-LD: %w", err)
	}

	return string(bytes), nil
}
