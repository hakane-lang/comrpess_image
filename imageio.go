package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// LoadImage decodes an image from disk, supporting JPEG, PNG, GIF and WebP.
func LoadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}

// SaveImage encodes img to output, choosing the codec based on file extension.
func SaveImage(img image.Image, output string, quality int) error {
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(output))

	switch ext {

	case ".jpg", ".jpeg":
		return jpeg.Encode(file, img, &jpeg.Options{
			Quality: quality,
		})

	case ".png":
		encoder := png.Encoder{
			CompressionLevel: png.BestCompression,
		}
		return encoder.Encode(file, img)

	case ".webp":
		return webp.Encode(file, img, &webp.Options{
			Lossless: false,
			Quality:  float32(quality),
		})

	default:
		return fmt.Errorf("unsupported output format: %s", ext)
	}
}

// fileSize returns the size in bytes of the file at path.
func fileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// humanSize formats a byte count as a human-readable string (e.g. "1.34 MiB").
func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
