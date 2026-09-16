package models

import (
	"errors"
	"time"
)

func isStaticPageThere(staticPageID string) error {
	// Check page existence
	exists, err := RecordExists(DB, "static_pages", "static_page_id = ?", staticPageID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("static page not found")
	}
	return nil
}

// FormatDateTimeString formats datetime strings to a standardized format.
//
// This utility function attempts to parse datetime strings from multiple common formats
// and converts them to a consistent "2006-01-02 15:04:05" format for display.
//
// Supported input formats:
//   - RFC3339: "2006-01-02T15:04:05Z07:00"
//   - MySQL datetime: "2006-01-02 15:04:05"
//   - ISO datetime: "2006-01-02T15:04:05"
//   - Date only: "2006-01-02"
//
// Parameters:
//   - dt: string - The datetime string to format
//
// Returns:
//   - string: Formatted datetime as "2006-01-02 15:04:05", or original string if parsing fails,
//     or empty string if input is empty
func FormatDateTimeString(dt string) string {
	// Return empty for empty input
	if dt == "" {
		return ""
	}

	// Define supported datetime formats to try
	layouts := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
		"2006-01-02 15:04:05", // MySQL datetime
		"2006-01-02T15:04:05", // ISO datetime without timezone
		"2006-01-02",          // Date only
	}

	var t time.Time
	var err error

	// Try parsing with each layout until one succeeds
	for _, layout := range layouts {
		t, err = time.Parse(layout, dt)
		if err == nil {
			// Successfully parsed - format to standard output with seconds
			return t.Format("2006-01-02 15:04:05")
		}
	}

	// If no format matched, return original string as fallback
	return dt
}
