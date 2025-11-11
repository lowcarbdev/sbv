//go:build heic

package internal


import (
	"bytes"
	"image/jpeg"

	"github.com/strukturag/libheif-go"
)

// convertHEICtoJPEG converts HEIC image data to JPEG format
// Returns the converted JPEG data or an error if conversion fails
// This version requires the libheif library and is enabled with the 'heic' build tag
func convertHEICtoJPEG(heicData []byte) ([]byte, error) {
	// Create a new HEIF context
	ctx, err := libheif.NewContext()
	if err != nil {
		return nil, err
	}

	// Read HEIC data from memory
	err = ctx.ReadFromMemory(heicData)
	if err != nil {
		return nil, err
	}

	// Get the primary image handle
	handle, err := ctx.GetPrimaryImageHandle()
	if err != nil {
		return nil, err
	}

	// Decode the image to RGB format
	img, err := handle.DecodeImage(libheif.ColorspaceRGB, libheif.ChromaInterleavedRGB, nil)
	if err != nil {
		return nil, err
	}

	// Convert to Go's standard image.Image
	goImg, err := img.GetImage()
	if err != nil {
		return nil, err
	}

	// Encode as JPEG with high quality
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, goImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
