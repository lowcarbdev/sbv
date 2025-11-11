//go:build !heic

package internal


import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log/slog"
)

// convertHEICtoJPEG returns a placeholder image when HEIC support is disabled
// This version does not require the libheif library
func convertHEICtoJPEG(heicData []byte) ([]byte, error) {
	slog.Warn("HEIC conversion is disabled. Returning placeholder image. Build with -tags heic to enable HEIC support.")

	// Return a simple placeholder JPEG image (400x300 gray rectangle with text)
	// This is better than returning an error, as it allows the app to function
	return generatePlaceholderJPEG()
}

// generatePlaceholderJPEG creates a simple gray placeholder image
func generatePlaceholderJPEG() ([]byte, error) {
	// Create a 400x300 image
	width, height := 400, 300
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with gray background
	gray := color.RGBA{200, 200, 200, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, gray)
		}
	}

	// Add a dark border
	borderColor := color.RGBA{100, 100, 100, 255}
	for x := 0; x < width; x++ {
		img.Set(x, 0, borderColor)
		img.Set(x, height-1, borderColor)
	}
	for y := 0; y < height; y++ {
		img.Set(0, y, borderColor)
		img.Set(width-1, y, borderColor)
	}

	// Encode as JPEG
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	if err != nil {
		return nil, fmt.Errorf("failed to encode placeholder image: %w", err)
	}

	return buf.Bytes(), nil
}

// Alternative: Return a base64-encoded minimal JPEG (1x1 pixel)
// This is more efficient but less user-friendly
func generateMinimalPlaceholderJPEG() ([]byte, error) {
	// 1x1 gray pixel JPEG (base64 encoded minimal JPEG)
	minimalJPEG := "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwA/wA/h"
	return base64.StdEncoding.DecodeString(minimalJPEG)
}
