package cover

import (
	"archive/zip"
	"io"
	"mime"
	"path/filepath"
	"sort"
	"strings"
)

func getCbzCover(filePath string) ([]byte, string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	var images []string
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
			images = append(images, f.Name)
		}
	}

	if len(images) == 0 {
		return nil, "", io.EOF // No images found
	}

	sort.Strings(images) // Alphabetical order, usually first image is cover

	coverPath := images[0]
	// Check if there is a file named "cover.jpg" or similar?
	// Common convention is that the first file alphabetically is the first page/cover.

	f, err := findFileInZip(r, coverPath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}

	mimeType := mime.TypeByExtension(filepath.Ext(coverPath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return data, mimeType, nil
}
