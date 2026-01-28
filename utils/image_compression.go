// Package utils provides image compression utilities
package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"strings"

	"golang.org/x/image/draw"
)

// CompressImageOptions defines options for image compression
type CompressImageOptions struct {
	MaxWidth       int  // Maximum width (0 = no limit)
	MaxHeight      int  // Maximum height (0 = no limit)
	Quality        int  // JPEG quality (1-100, default 85)
	PreserveFormat bool // If true, keep original format; if false, convert to JPEG
}

// CompressImage compresses and optionally resizes an image from a multipart file header
//
// This function:
// 1. Detects image format (JPEG, PNG, etc.)
// 2. Decodes the image
// 3. Resizes if dimensions exceed max width/height
// 4. Compresses with specified quality
// 5. Returns compressed image as bytes
//
// Parameters:
//   - fileHeader: *multipart.FileHeader - The uploaded file
//   - opts: CompressImageOptions - Compression options
//
// Returns:
//   - []byte: Compressed image data
//   - string: MIME type of compressed image
//   - error: Any error during processing
func CompressImage(fileHeader *multipart.FileHeader, opts CompressImageOptions) ([]byte, string, error) {
	// Set defaults
	if opts.Quality == 0 {
		opts.Quality = 85
	}
	if opts.MaxWidth == 0 {
		opts.MaxWidth = 2048 // Default max width
	}
	if opts.MaxHeight == 0 {
		opts.MaxHeight = 2048 // Default max height
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Detect image format from MIME type
	mimeType := fileHeader.Header.Get("Content-Type")
	isImage := strings.HasPrefix(mimeType, "image/")

	// If not an image, return original data without compression
	if !isImage {
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, file); err != nil {
			return nil, "", fmt.Errorf("failed to read file: %w", err)
		}
		return buf.Bytes(), mimeType, nil
	}

	// Decode the image (supports JPEG, PNG, GIF, etc.)
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Get original dimensions
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Calculate new dimensions if resizing is needed
	newWidth, newHeight := origWidth, origHeight
	if origWidth > opts.MaxWidth || origHeight > opts.MaxHeight {
		ratio := float64(origWidth) / float64(origHeight)

		if origWidth > opts.MaxWidth {
			newWidth = opts.MaxWidth
			newHeight = int(float64(newWidth) / ratio)
		}

		if newHeight > opts.MaxHeight {
			newHeight = opts.MaxHeight
			newWidth = int(float64(newHeight) * ratio)
		}
	}

	// Resize if dimensions changed
	var finalImg image.Image
	if newWidth != origWidth || newHeight != origHeight {
		resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)
		finalImg = resized
	} else {
		finalImg = img
	}

	// Encode the image
	buf := new(bytes.Buffer)
	var outputMimeType string

	if opts.PreserveFormat && (format == "jpeg" || format == "jpg") {
		// Encode as JPEG with specified quality
		err = jpeg.Encode(buf, finalImg, &jpeg.Options{Quality: opts.Quality})
		outputMimeType = "image/jpeg"
	} else if opts.PreserveFormat && format == "png" {
		// Encode as PNG
		err = png.Encode(buf, finalImg)
		outputMimeType = "image/png"
	} else {
		// Default: convert to JPEG for maximum compression
		err = jpeg.Encode(buf, finalImg, &jpeg.Options{Quality: opts.Quality})
		outputMimeType = "image/jpeg"
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), outputMimeType, nil
}
