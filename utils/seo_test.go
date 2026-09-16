package utils

import (
	"encoding/json"
	"testing"
)

func TestGenerateProductJSONLD(t *testing.T) {
	params := ProductJSONLDParams{
		Name:        "Test Product",
		Image:       "https://example.com/image.jpg",
		Description: "A great product",
		SKU:         "SKU123",
		Currency:    "USD",
		Price:       99.99,
		InStock:     true,
		ProductURL:  "https://example.com/product/123",
	}

	result, err := GenerateProductJSONLD(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var schema ProductSchemaLD
	if err := json.Unmarshal([]byte(result), &schema); err != nil {
		t.Fatalf("failed to unmarshal JSON-LD output: %v", err)
	}

	if schema.Name != params.Name {
		t.Errorf("expected Name %q, got %q", params.Name, schema.Name)
	}
	if schema.Offers.Price != params.Price {
		t.Errorf("expected Price %f, got %f", params.Price, schema.Offers.Price)
	}
	if schema.Offers.Availability != "https://schema.org/InStock" {
		t.Errorf("expected InStock availability, got %q", schema.Offers.Availability)
	}
}

func TestGenerateProductJSONLD_DefaultsAndOutOfStock(t *testing.T) {
	params := ProductJSONLDParams{
		Name:    "Out of Stock Item",
		Price:   10.0,
		InStock: false,
	}

	result, err := GenerateProductJSONLD(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var schema ProductSchemaLD
	if err := json.Unmarshal([]byte(result), &schema); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if schema.Offers.PriceCurrency != "KES" {
		t.Errorf("expected default currency KES, got %q", schema.Offers.PriceCurrency)
	}
	if schema.Offers.Availability != "https://schema.org/OutOfStock" {
		t.Errorf("expected OutOfStock availability, got %q", schema.Offers.Availability)
	}
}
