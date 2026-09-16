package handlers

import (
	"database/sql"
	"ekomasi_backend/models"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// setupTestDB replaces the global models.DB with a mock for testing purposes.
// It returns the abstract sql.DB, the mock controller, and a teardown function.
func setupTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	// Create mock DB connection
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	// Save the original DB instance
	originalDB := models.DB

	// Replace global DB with mock
	models.DB = db

	// Return mock and cleanup function
	return db, mock, func() {
		// Restore original DB and close mock
		models.DB = originalDB
		db.Close()
	}
}
