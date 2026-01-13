package utils

import (
	"fmt"
	"log"
	"mime"
	"opds-server/internal/models"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FormatDate formats a time.Time into a readable date string
func FormatDate(t time.Time) string {
	return t.Format("Jan 02, 2006 15:04")
}

// FormatSize converts file size in bytes to a readable format
func FormatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// CleanupTitle cleans up the filename to create a readable title
func CleanupTitle(filename string) string {
	// Remove file extension
	title := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Replace common separators with spaces
	title = strings.NewReplacer("-", " ", "_", " ").Replace(title)
	return strings.TrimSpace(title)
}

// GetBooksList gets list of books from the specified directory recursively
// SortBooks sorts a slice of BookInfo based on the provided mode
func SortBooks(books []models.BookInfo, mode string) {
	switch mode {
	case "name-asc":
		sort.Slice(books, func(i, j int) bool {
			return strings.ToLower(books[i].Title) < strings.ToLower(books[j].Title)
		})
	case "name-desc":
		sort.Slice(books, func(i, j int) bool {
			return strings.ToLower(books[i].Title) > strings.ToLower(books[j].Title)
		})
	case "date-asc":
		sort.Slice(books, func(i, j int) bool {
			return books[i].LastUpdated.Before(books[j].LastUpdated)
		})
	case "date-desc":
		sort.Slice(books, func(i, j int) bool {
			return books[i].LastUpdated.After(books[j].LastUpdated)
		})
	default:
		// Default to date-desc (newest first)
		sort.Slice(books, func(i, j int) bool {
			return books[i].LastUpdated.After(books[j].LastUpdated)
		})
	}
}

func GetBooksList(booksDir string) ([]models.BookInfo, error) {
	var books []models.BookInfo

	// Supported extensions
	var validExt = map[string]struct{}{
		".epub": {},
		".pdf":  {},
		".fb2":  {},
		".mobi": {},
		".azw":  {},
		".azw3": {},
		".azw4": {},
		".txt":  {},
		".rtf":  {},
		".html": {},
		".htm":  {},
		".djvu": {},
		".cbz":  {},
		".cbr":  {},
		".cb7":  {},
	}

	err := filepath.WalkDir(booksDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			log.Printf("Error accessing path %q: %v", path, err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		fileInfo, err := d.Info()
		if err != nil {
			log.Printf("Error getting info for %s: %v", d.Name(), err)
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := validExt[ext]; !ok {
			return nil
		}

		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		relPath, err := filepath.Rel(booksDir, path)
		if err != nil {
			return err
		}
		// Ensure forward slashes for URL usage
		webPath := filepath.ToSlash(relPath)

		books = append(books, models.BookInfo{
			Filename:    webPath,
			Title:       CleanupTitle(d.Name()),
			MimeType:    mimeType,
			LastUpdated: fileInfo.ModTime().UTC(),
			Size:        fileInfo.Size(),
		})
		return nil
	})

	return books, err
}
