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
	"google.golang.org/api/option"
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

// UploadMediaToGCS uploads a single file to Google Cloud Storage and returns the public URL
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

// UploadMediaToGCS uploads the media data to Google Cloud Storage
func uploadMedia(mediaData []byte, mediaID, mimeType string) (string, error) {
	log.Println("Starting upload to GCS")
	// Google Cloud Storage bucket name
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		bucketName = "ecommerce-adenzo"
	}
	// Determine the file extension from the MIME type
	extension := strings.Split(mimeType, "/")[1]
	// Generate a unique object name using the media ID and a unique ID
	objectName := fmt.Sprintf("attachments/%s_%s.%s", mediaID, generateUniqueID(), extension)
	ctx := context.Background()

	// Create a new Google Cloud Storage client
	// serviceAccount := os.Getenv("SERVICE_ACCOUNT")
	// commented out this so as to await infra team to set up bucket service to upload images
	serviceAccount := ""
	log.Println("Creating storage client")
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(serviceAccount))
	if err != nil {
		log.Printf("Failed to create storage client: %v", err)
		return "", fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	// Get a handle to the bucket and the object
	bucket := client.Bucket(bucketName)
	object := bucket.Object(objectName)

	// Create a new writer for the object
	log.Println("Creating writer for the object")
	wc := object.NewWriter(ctx)
	// Write the media data to the object
	log.Println("Uploading media data")
	if _, err := io.Copy(wc, bytes.NewReader(mediaData)); err != nil {
		log.Printf("Failed to upload media data: %v", err)
		return "", fmt.Errorf("failed to upload media data: %v", err)
	}
	// Close the writer
	log.Println("Closing writer")
	if err := wc.Close(); err != nil {
		log.Printf("Failed to close writer: %v", err)
		return "", fmt.Errorf("failed to close writer: %v", err)
	}

	// Make the object public by setting the ACL
	log.Println("Setting object ACL to public")
	if err := object.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		log.Printf("Failed to make object public: %v", err)
		return "", fmt.Errorf("failed to make object public: %v", err)
	}

	// Generate the public URL
	url := fmt.Sprintf("https://bucket.emalify.com/%s", objectName)
	log.Printf("File uploaded to GCS and made public: %s", objectName)
	return url, nil
}

// generateUniqueID generates a unique ID based on the current time in nanoseconds.
func generateUniqueID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
