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

// ParseAndUploadFile handles file extraction and upload to GCS.
// formFieldName is the name of the form input (e.g., "image" or "file").
// maxMemoryMB defines the maximum allowed memory for parsing multipart form data.
func ParseAndUploadFile(r *http.Request, formFieldName string, maxMemoryMB int64) (string, error) {
	// Parse multipart form
	if err := r.ParseMultipartForm(maxMemoryMB << 20); err != nil {
		return "", err
	}

	// Retrieve file
	file, header, err := r.FormFile(formFieldName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Upload to GCS
	url, err := UploadMediaToGCS([]*multipart.FileHeader{header})
	if err != nil {
		return "", err
	}

	return url, nil
}

// generateUniqueID generates a unique ID based on the current time in nanoseconds.
func generateUniqueID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func uploadMedia(mediaData []byte, mediaID, mimeType string) (string, error) {
	log.Println("Starting upload to GCS")

	// Bucket name from env or default
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		bucketName = "development-ecommerce-api-images"
	}
	log.Printf("Using bucket name:::: %s", bucketName)
	// File extension from MIME type
	ext := "bin"
	if parts := strings.Split(mimeType, "/"); len(parts) > 1 {
		ext = parts[1]
	}

	objectName := fmt.Sprintf("attachments/%s_%s.%s", mediaID, generateUniqueID(), ext)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	// Object handle with precondition: fails if object already exists
	o := client.Bucket(bucketName).Object(objectName).If(storage.Conditions{DoesNotExist: true})

	// Use a bytes.Reader instead of a local file
	wc := o.NewWriter(ctx)
	if _, err := io.Copy(wc, bytes.NewReader(mediaData)); err != nil {
		return "", fmt.Errorf("io.Copy: %w", err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("Writer.Close: %w", err)
	}

	// Optional: public access (remove if using uniform bucket-level access)
	if err := o.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		log.Printf("Cannot set ACL: %v", err)
	}

	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName)
	log.Printf("Uploaded to %s", publicURL)
	return publicURL, nil
}

func UploadMediaToGCS(files []*multipart.FileHeader) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files provided")
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		log.Printf("Unable to open file: %v", err)
		return "", fmt.Errorf("unable to open file: %v", err)
	}
	defer file.Close()

	// Read the file content into a buffer
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		log.Printf("Unable to read file: %v", err)
		return "", fmt.Errorf("unable to read file: %v", err)
	}

	// Generate a unique ID for the media and upload it to GCS
	mediaID := generateUniqueID()
	mimeType := fileHeader.Header.Get("Content-Type")

	url, err := uploadMedia(buf.Bytes(), mediaID, mimeType)
	if err != nil {
		log.Printf("Unable to upload file to GCS: %v", err)
		return "", fmt.Errorf("unable to upload file to GCS: %v", err)
	}

	return url, nil
}
