package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	input := flag.String("in", "", "Input image")
	output := flag.String("out", "", "Output image")
	quality := flag.Int("quality", 90, "JPEG/WebP quality (0-100)")

	flag.Parse()

	// Scriptable one-shot mode: used whenever at least one flag is passed.
	if *input != "" || *output != "" {
		if *input == "" || *output == "" {
			fmt.Println("Usage:")
			fmt.Println("  compress -in input.jpg -out output.webp")
			fmt.Println("  compress -in input.png -out output.jpg -quality 80")
			os.Exit(1)
		}

		if *quality < 0 || *quality > 100 {
			fmt.Println("quality must be in range 0-100")
			os.Exit(1)
		}

		if err := runOnce(*input, *output, *quality); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		return
	}

	// Interactive mode: a friendly, looping terminal UI.
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func runOnce(in, out string, quality int) error {
	img, err := LoadImage(in)
	if err != nil {
		return err
	}

	if err := SaveImage(img, out, quality); err != nil {
		return err
	}

	fmt.Println("✓ Saved:", out)
	return nil
}
