package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"

	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

func main() {
	input := flag.String("in", "", "Input image")
	output := flag.String("out", "", "Output image")
	quality := flag.Int("quality", 90, "JPEG/WebP quality (0-100)")

	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Println("Usage:")
		fmt.Println("  Compress -in input.jpg -out output.webp")
		fmt.Println("  Compress -in input.png -out output.jpg -quality 80")
		os.Exit(1)
	}

	if *quality<0 || *quality>100{
		fmt.Println("quality in range 0-100")
		os.Exit(1)
	}

	img, err := LoadImage(*input)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	err = SaveImage(img, *output, *quality)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("✓ Saved:", *output)
}

func LoadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}

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