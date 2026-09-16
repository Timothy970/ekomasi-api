package utils

import (
	"testing"
	"time"

	"ekomasi_backend/dtos"
)

func TestEllipseText(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Short text", 20, "Short text"},
		{"Exact length text", 17, "Exact length text"},
		{"This is a long description that needs truncation", 20, "This is a long de..."},
		{"ABC", 2, "AB"},
	}

	for _, tt := range tests {
		result := EllipseText(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("EllipseText(%q, %d) = %q; expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestGetImageFormatFromContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"image/jpeg", "JPEG"},
		{"image/jpg", "JPEG"},
		{"image/png", "PNG"},
		{"image/webp", "WEBP"},
		{"image/avif", "AVIF"},
		{"unknown/type", "PNG"},
	}

	for _, tt := range tests {
		result := getImageFormatFromContentType(tt.contentType)
		if result != tt.expected {
			t.Errorf("getImageFormatFromContentType(%q) = %q; expected %q", tt.contentType, result, tt.expected)
		}
	}
}

func TestGenerateInvoicePDF(t *testing.T) {
	deliveryCharge := 150.0
	order := dtos.AdminOrder{
		OrderID:        "ORD-12345",
		OrderStatus:    "PAID",
		CreatedAt:      time.Now(),
		TotalAmount:    1500.0,
		SubTotal:       1250.0,
		EstimatedTax:   100.0,
		TotalDiscount:  0.0,
		DeliveryCharge: &deliveryCharge,
		PaymentMethod:  "M-PESA",
		PaymentStatus:  "Completed",
		User: &dtos.Users{
			FirstName: "Jane",
			LastName:  "Doe",
			Email:     "jane@example.com",
			Phone:     "+254712345678",
		},
		Items: []dtos.OrderProduct{
			{
				Description: "Wireless Mouse Ergonomic Design",
				Price:       1250.0,
				Images: []dtos.Image{
					{URL: ""},
				},
			},
		},
	}

	pdfBytes, err := GenerateInvoicePDF(order)
	if err != nil {
		t.Fatalf("GenerateInvoicePDF returned error: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Errorf("GenerateInvoicePDF returned empty byte slice")
	}
}
