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
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strings"
)

// UploadMediaToGCS uploads media files to Google Cloud Storage from multipart file headers.
//
// This function:
// 1. Validates at least one file is provided
// 2. Compresses images to reduce file size (quality: 85%, max: 2048x2048)
// 3. Reads file content into memory buffer
// 4. Generates unique media ID
// 5. Extracts MIME type from file header
// 6. Uploads to GCS and returns public URL
//
// File Processing:
//   - Only processes first file in array
//   - Automatically compresses image files before upload
//   - Non-image files are uploaded as-is
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

	// Extract MIME type from file header
	mimeType := fileHeader.Header.Get("Content-Type")

	var fileData []byte
	var finalMimeType string

	// Check if file is an image and compress it
	if strings.HasPrefix(mimeType, "image/") {
		log.Printf("Compressing image: %s (original MIME: %s)", fileHeader.Filename, mimeType)

		// Compress the image
		compressedData, compressedMimeType, err := CompressImage(fileHeader, CompressImageOptions{
			MaxWidth:       2048,
			MaxHeight:      2048,
			Quality:        85,
			PreserveFormat: false, // Convert to JPEG for better compression
		})

		if err != nil {
			log.Printf("Failed to compress image, uploading original: %v", err)
			// If compression fails, fall back to original file
			file, err := fileHeader.Open()
			if err != nil {
				log.Printf("Unable to open file: %v", err)
				return "", fmt.Errorf("unable to open file: %v", err)
			}
			defer file.Close()

			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, file); err != nil {
				log.Printf("Unable to read file: %v", err)
				return "", fmt.Errorf("unable to read file: %v", err)
			}
			fileData = buf.Bytes()
			finalMimeType = mimeType
		} else {
			log.Printf("Image compressed successfully. Original size: %d bytes, Compressed size: %d bytes",
				fileHeader.Size, len(compressedData))
			fileData = compressedData
			finalMimeType = compressedMimeType
		}
	} else {
		// Non-image file - upload as-is
		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("Unable to open file: %v", err)
			return "", fmt.Errorf("unable to open file: %v", err)
		}
		defer file.Close()

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, file); err != nil {
			log.Printf("Unable to read file: %v", err)
			return "", fmt.Errorf("unable to read file: %v", err)
		}
		fileData = buf.Bytes()
		finalMimeType = mimeType
	}

	// Generate unique ID for the media file
	mediaID := generateUniqueID()

	// Upload file bytes to GCS
	url, err := uploadMedia(fileData, mediaID, finalMimeType)
	if err != nil {
		return "", fmt.Errorf("unable to upload file to GCS: %v", err)
	}

	// Return public URL of uploaded file
	return url, nil
}
