package logger

import (
	"adenzo_backend/models"
	"encoding/json"
	"log"

	"github.com/teris-io/shortid"
)

type LogLevel string

const (
	InfoLevel  LogLevel = "INFO"
	ErrorLevel LogLevel = "ERROR"
	WarnLevel  LogLevel = "WARN"
	DebugLevel LogLevel = "DEBUG"
)

// LogEntry represents a structured log
type LogEntry struct {
	Level    LogLevel
	Message  string
	UserID   *string // nullable
	Metadata map[string]interface{}
}

// Log writes the log entry to the logs table
func Log(entry LogEntry) {
	logID, _ := shortid.Generate()
	meta, _ := json.Marshal(entry.Metadata)

	_, err := models.DB.Exec(`
		INSERT INTO logs (log_id, level, message, user_id, metadata)
		VALUES (?, ?, ?, ?, ?)`,
		logID, entry.Level, entry.Message, entry.UserID, string(meta),
	)

	if err != nil {
		log.Printf("Failed to insert log: %v", err)
	}
}
