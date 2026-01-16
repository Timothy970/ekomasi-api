// Package utils provides image and media upload utilities for the Adenzo e-commerce platform.
//
// This file contains Google Cloud Storage (GCS) upload functions:
//   - Multipart form file parsing and extraction
//   - Media upload to GCS buckets
//   - Public URL generation for uploaded files
//   - Unique file naming with timestamp-based IDs
//   - MIME type detection and file extension handling
//
// Upload Features:
//   - Direct upload to Google Cloud Storage
//   - Automatic file naming with unique IDs
//   - Public access control (ACL) configuration
//   - Memory-efficient streaming uploads
//   - Timeout protection (50 seconds)
//   - Precondition checks to prevent overwrites
//
// File Handling:
//   - Multipart form data parsing
//   - File size limits (configurable max memory)
//   - MIME type based file extension detection
//   - Bytes buffer for in-memory processing
//
// GCS Configuration:
//   - Bucket name from environment variable (BUCKET_NAME)
//   - Default bucket: "development-ecommerce-api-images"
//   - Storage path: "attachments/<mediaID>_<timestamp>.<ext>"
//   - Public URL format: https://storage.googleapis.com/<bucket>/<object>
package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

// ParseAndUploadFile handles file extraction from HTTP request and upload to Google Cloud Storage.
//
// This function:
// 1. Parses multipart form data from HTTP request
// 2. Extracts file from specified form field
// 3. Uploads file to GCS bucket
// 4. Returns public URL of uploaded file
//
// Memory Management:
//   - maxMemoryMB limits memory used for form parsing
//   - Remaining data stored in temporary files
//   - File automatically closed after processing
//
// Parameters:
//   - r: *http.Request - HTTP request containing multipart form data
//   - formFieldName: string - Name of the form input field (e.g., "image", "file")
//   - maxMemoryMB: int64 - Maximum memory in MB for parsing (e.g., 10 for 10MB)
//
// Returns:
//   - string: Public URL of uploaded file in GCS
//   - error: Parse error, file extraction error, or upload error
func ParseAndUploadFile(r *http.Request, formFieldName string, maxMemoryMB int64) (string, error) {
	// Parse multipart form with memory limit (convert MB to bytes)
	if err := r.ParseMultipartForm(maxMemoryMB << 20); err != nil {
		return "", err
	}

	// Retrieve file from form field
	file, header, err := r.FormFile(formFieldName)
	if err != nil {
		return "", err
	}
	defer file.Close() // Ensure file is closed after processing

	// Upload file to Google Cloud Storage
	url, err := UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		return "", err
	}

	// Return public URL of uploaded file
	return url, nil
}

// generateUniqueID generates a unique identifier based on current time in nanoseconds.
//
// This function:
// 1. Gets current time in nanoseconds since Unix epoch
// 2. Converts to string format
// 3. Returns as unique identifier
//
// Uniqueness:
//   - Based on nanosecond precision timestamp
//   - Effectively unique for sequential calls
//   - Not guaranteed unique for parallel calls
//
// Returns:
//   - string: Unique ID in format "1234567890123456789"
func generateUniqueID() string {
	// Use nanosecond timestamp for uniqueness
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// uploadMedia uploads media data to Google Cloud Storage bucket.
//
// This function:
// 1. Determines bucket name from environment or uses default
// 2. Extracts file extension from MIME type
// 3. Generates unique object name with timestamp
// 4. Creates GCS client with timeout
// 5. Uploads data with precondition (no overwrite)
// 6. Sets public read access (ACL)
// 7. Returns public URL
//
// Bucket Configuration:
//   - Environment variable: BUCKET_NAME
//   - Default: "development-ecommerce-api-images"
//
// Object Naming:
//   - Format: "attachments/<mediaID>_<timestamp>.<ext>"
//
// Upload Settings:
//   - Timeout: 50 seconds
//   - Precondition: DoesNotExist (prevents overwrites)
//   - ACL: AllUsers with Reader role (public access)
//
// Parameters:
//   - mediaData: []byte - File content as byte array
//   - mediaID: string - Base identifier for the media file
//   - mimeType: string - MIME type (e.g., "image/jpeg", "application/pdf")
//
// Returns:
//   - string: Public URL of uploaded file
//   - error: Client creation, upload, or ACL setting error
func uploadMedia(mediaData []byte, mediaID, mimeType string) (string, error) {
	log.Println("Starting upload to GCS")

	// Get bucket name from environment or use default
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		bucketName = "development-ecommerce-api-images"
	}
	log.Printf("Using bucket name:::: %s", bucketName)

	// Extract file extension from MIME type (e.g., "image/jpeg" -> "jpeg")
	ext := "bin" // Default extension for unknown types
	if parts := strings.Split(mimeType, "/"); len(parts) > 1 {
		ext = parts[1] // Use second part as extension
	}

	// Generate unique object name with path prefix
	objectName := fmt.Sprintf("attachments/%s_%s.%s", mediaID, generateUniqueID(), ext)

	// Create context with 50 second timeout
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel() // Ensure timeout is cancelled

	// Create GCS client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close() // Ensure client is closed

	// Get object handle with precondition: fails if object already exists
	o := client.Bucket(bucketName).Object(objectName).If(storage.Conditions{DoesNotExist: true})

	// Create writer for uploading data
	wc := o.NewWriter(ctx)
	// Stream data from bytes buffer to GCS
	if _, err := io.Copy(wc, bytes.NewReader(mediaData)); err != nil {
		return "", fmt.Errorf("io.Copy: %w", err)
	}

	// Close writer to finalize upload
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("Writer.Close: %w", err)
	}

	// Set public read access (optional - remove if using uniform bucket-level access)
	if err := o.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		log.Printf("Cannot set ACL: %v", err)
	}

	// Construct public URL for the uploaded file
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName)
	log.Printf("Uploaded to %s", publicURL)
	return publicURL, nil
}

// UploadMediaToGCS uploads media files to Google Cloud Storage from multipart file headers.
//
// This function:
// 1. Validates at least one file is provided
// 2. Opens the first file from the array
// 3. Reads file content into memory buffer
// 4. Generates unique media ID
// 5. Extracts MIME type from file header
// 6. Uploads to GCS and returns public URL
//
// File Processing:
//   - Only processes first file in array
//   - Reads entire file into memory buffer
//   - Uses file header Content-Type for MIME detection
//   - Automatic file closure after reading
//
// Parameters:
//   - files: []*multipart.FileHeader - Array of multipart file headers
//     (only first file is processed)
//
// Returns:
//   - string: Public URL of uploaded file in GCS
//   - error: No files, file open error, read error, or upload error
func UploadMediaToGCS(files []*multipart.FileHeader) (string, error) {
	// Validate files array is not empty
	if len(files) == 0 {
		return "", fmt.Errorf("no files provided")
	}

	// Get first file header from array
	fileHeader := files[0]
	// Open file for reading
	file, err := fileHeader.Open()
	if err != nil {
		log.Printf("Unable to open file: %v", err)
		return "", fmt.Errorf("unable to open file: %v", err)
	}
	defer file.Close() // Ensure file is closed

	// Read entire file content into memory buffer
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		log.Printf("Unable to read file: %v", err)
		return "", fmt.Errorf("unable to read file: %v", err)
	}

	// Generate unique ID for the media file
	mediaID := generateUniqueID()
	// Extract MIME type from file header
	mimeType := fileHeader.Header.Get("Content-Type")

	// Upload file bytes to GCS
	url, err := uploadMedia(buf.Bytes(), mediaID, mimeType)
	if err != nil {
		log.Printf("Unable to upload file to GCS: %v", err)
		return "", fmt.Errorf("unable to upload file to GCS: %v", err)
	}

	// Return public URL of uploaded file
	return url, nil
}
