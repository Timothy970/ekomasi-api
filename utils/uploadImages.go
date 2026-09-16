// Package utils provides image and media upload utilities for the Ekomasi e-commerce platform.
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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ekomasi_backend/config"
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

// NewStorageClient initializes a storage client based on application environment.
// If FLOCI_ENDPOINT is configured, it connects to local Floci / emulator without authentication.
// Otherwise, it creates a standard Google Cloud Storage client using Application Default Credentials.
func NewStorageClient(ctx context.Context) (*storage.Client, error) {
	flociEndpoint := os.Getenv("FLOCI_ENDPOINT")
	if flociEndpoint == "" && config.Get() != nil {
		flociEndpoint = config.Get().Storage.FlociEndpoint
	}

	if flociEndpoint != "" {
		log.Printf("Connecting to local storage endpoint (Floci): %s", flociEndpoint)
		return storage.NewClient(
			ctx,
			option.WithEndpoint(flociEndpoint),
			option.WithoutAuthentication(),
		)
	}

	return storage.NewClient(ctx)
}

// ensureBucketExists automatically creates the bucket on local Floci emulator if it doesn't already exist.
func ensureBucketExists(ctx context.Context, client *storage.Client, bucketName string) error {
	flociEndpoint := os.Getenv("FLOCI_ENDPOINT")
	if flociEndpoint == "" && config.Get() != nil {
		flociEndpoint = config.Get().Storage.FlociEndpoint
	}

	// Only auto-create bucket when running against local Floci endpoint
	if flociEndpoint != "" {
		projectID := os.Getenv("GCP_PROJECT_ID")
		if projectID == "" {
			projectID = "local-project"
		}

		bucket := client.Bucket(bucketName)
		if err := bucket.Create(ctx, projectID, nil); err != nil {
			if status.Code(err) != codes.AlreadyExists {
				log.Printf("Floci bucket auto-creation notice: %v", err)
			}
		}
	}
	return nil
}

// publicURL returns the public access URL for an uploaded object.
// Returns local endpoint URL format if FLOCI_ENDPOINT is configured, else standard GCP storage URL.
func publicURL(bucketName, objectName string) string {
	flociEndpoint := os.Getenv("FLOCI_ENDPOINT")
	if flociEndpoint == "" && config.Get() != nil {
		flociEndpoint = config.Get().Storage.FlociEndpoint
	}

	if flociEndpoint != "" {
		return fmt.Sprintf(
			"%s/%s/%s",
			strings.TrimSuffix(flociEndpoint, "/"),
			bucketName,
			objectName,
		)
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName)
}

// uploadMedia uploads media data to storage bucket (Floci or GCP Cloud Storage).
//
// This function:
// 1. Determines bucket name from environment or configuration
// 2. Extracts file extension from MIME type
// 3. Generates unique object name with timestamp
// 4. Creates storage client based on environment (Floci vs GCS)
// 5. Uploads data with precondition (no overwrite)
// 6. Sets public read access (ACL)
// 7. Returns public URL
//
// Bucket Configuration:
//   - Environment variables: STORAGE_BUCKET or BUCKET_NAME
//   - Default: "development-ecommerce-api-images"
//
// Object Naming:
//   - Format: "attachments/<mediaID>_<timestamp>.<ext>"
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
	log.Println("Starting upload to storage")

	// Check if Cloudinary upload is enabled
	useCloudinary := os.Getenv("USE_CLOUDINARY") == "true"
	if !useCloudinary && config.Get() != nil {
		useCloudinary = config.Get().Storage.UseCloudinary
	}

	if useCloudinary {
		log.Println("USE_CLOUDINARY is enabled. Uploading image to Cloudinary...")
		return UploadToCloudinary(mediaData, mediaID, mimeType)
	}

	// Get bucket name from config / environment or use default
	bucketName := os.Getenv("STORAGE_BUCKET")
	if bucketName == "" {
		bucketName = os.Getenv("BUCKET_NAME")
	}
	if bucketName == "" && config.Get() != nil {
		bucketName = config.Get().Storage.Bucket
	}
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

	// Create context with 300 second (5 minute) timeout for large file uploads
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel() // Ensure timeout is cancelled

	// Create storage client based on environment
	client, err := NewStorageClient(ctx)
	if err != nil {
		return "", fmt.Errorf("NewStorageClient: %w", err)
	}
	defer client.Close() // Ensure client is closed

	// Auto-create bucket if running locally with Floci
	if err := ensureBucketExists(ctx, client, bucketName); err != nil {
		log.Printf("ensureBucketExists: %v", err)
	}

	// Get object handle with precondition: fails if object already exists
	o := client.Bucket(bucketName).Object(objectName).If(storage.Conditions{DoesNotExist: true})

	// Create writer for uploading data
	wc := o.NewWriter(ctx)
	// Stream data from bytes buffer to storage
	if _, err := io.Copy(wc, bytes.NewReader(mediaData)); err != nil {
		return "", fmt.Errorf("io.Copy: %w", err)
	}

	// Close writer to finalize upload
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("Writer.Close: %w", err)
	}

	// Set public read access
	if err := o.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		log.Printf("Cannot set ACL: %v", err)
	}

	// Construct public URL for the uploaded file
	url := publicURL(bucketName, objectName)
	log.Printf("Uploaded to %s", url)
	return url, nil
}

// UploadToCloudinary uploads media data to Cloudinary via HTTP REST API.
//
// API endpoint: https://api.cloudinary.com/v1_1/<CLOUD_NAME>/image/upload
// Auth: Basic Auth using API_KEY:API_SECRET (-u "<API_KEY>:<API_SECRET>")
// Fields: file, public_id
func UploadToCloudinary(mediaData []byte, mediaID, mimeType string) (string, error) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if (cloudName == "" || apiKey == "" || apiSecret == "") && config.Get() != nil {
		if cloudName == "" {
			cloudName = config.Get().Storage.CloudinaryCloudName
		}
		if apiKey == "" {
			apiKey = config.Get().Storage.CloudinaryAPIKey
		}
		if apiSecret == "" {
			apiSecret = config.Get().Storage.CloudinaryAPISecret
		}
	}

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return "", fmt.Errorf("cloudinary credentials missing (CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY, CLOUDINARY_API_SECRET)")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	ext := "jpg"
	if parts := strings.Split(mimeType, "/"); len(parts) > 1 {
		ext = parts[1]
	}
	filename := fmt.Sprintf("%s.%s", mediaID, ext)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file for Cloudinary: %w", err)
	}
	if _, err := part.Write(mediaData); err != nil {
		return "", fmt.Errorf("failed to write file data for Cloudinary: %w", err)
	}

	publicID := fmt.Sprintf("attachments/%s", mediaID)
	if err := writer.WriteField("public_id", publicID); err != nil {
		return "", fmt.Errorf("failed to write public_id field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", cloudName)
	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create Cloudinary HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send upload request to Cloudinary: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Cloudinary response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("cloudinary upload failed [status %d]: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SecureURL string `json:"secure_url"`
		URL       string `json:"url"`
		PublicID  string `json:"public_id"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal Cloudinary response JSON: %w", err)
	}

	if result.SecureURL != "" {
		log.Printf("Uploaded to Cloudinary (secure_url): %s", result.SecureURL)
		return result.SecureURL, nil
	}
	if result.URL != "" {
		log.Printf("Uploaded to Cloudinary (url): %s", result.URL)
		return result.URL, nil
	}

	return "", fmt.Errorf("cloudinary returned no URL: %s", string(respBody))
}
