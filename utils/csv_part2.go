// Package utils provides utility functions for the Ekomasi e-commerce platform.
//
// This file contains CSV processing utilities:
//   - Product bulk upload CSV parsing
//   - Inventory export to CSV format
//   - CSV validation and sanitization
//   - Type conversion helpers (string, float, pointer handling)
//   - Error reporting for invalid CSV data
//
// CSV Import Features:
//   - Header validation with expected column names
//   - Required field enforcement
//   - Data type conversion (string to int, float, bool)
//   - Row-level error handling (skip invalid, continue processing)
//   - Whitespace trimming and empty line skipping
//
// CSV Export Features:
//   - Multi-section CSV generation (Basic Info, Supplier Info, Additional Info)
//   - Null-safe pointer handling
//   - Number formatting
//   - Image URL aggregation
package utils

import (
	"strconv"
	"strings"
)

// ptrToStr safely converts string pointer to string.
//
// Returns empty string if pointer is nil, otherwise returns the dereferenced value.
//
// Parameters:
//   - s: *string - Pointer to string (may be nil)
//
// Returns:
//   - string: Dereferenced value or empty string
func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// floatToStr converts float to formatted string.
//
// Formats float with 2 decimal places for CSV export.
//
// Parameters:
//   - f: float64 - Float value to format
//
// Returns:
//   - string: Formatted float string (e.g., "123.45")
func floatToStr(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

// strToPtr converts string to string pointer.
//
// Returns nil if string is empty, otherwise returns pointer to string.
//
// Parameters:
//   - s: string - String value to convert
//
// Returns:
//   - *string: Pointer to string or nil if empty
func strToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// strToSlicePtr converts comma-separated string to string slice pointer.
//
// Returns nil if string is empty, otherwise splits by comma and returns pointer to string slice.
//
// Parameters:
//   - s: string - Comma-separated string value to convert
//
// Returns:
//   - *[]string: Pointer to string slice or nil if empty
func strToSlicePtr(s string) *[]string {
	if s == "" {
		return nil
	}
	items := strings.Split(s, ",")
	// Trim whitespace from each item
	for i := range items {
		items[i] = strings.TrimSpace(items[i])
	}
	return &items
}
