package cover

import (
	"errors"
	"path/filepath"
	"strings"
)

// GetCover extracts the cover image from a book file.
// It returns the image content, the mime type, and an error if any.
func GetCover(filePath string) ([]byte, string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".epub":
		return getEpubCover(filePath)
	case ".cbz":
		return getCbzCover(filePath)
	case ".fb2", ".fb2.zip": // .fb2.zip logic might need to be handled if inside zip
		if ext == ".zip" && strings.HasSuffix(strings.ToLower(filePath), ".fb2.zip") {
			// Basic extension check might fail for complex cases, but let's assume standard naming
			// actually filepath.Ext returns last extension.
			return getFb2Cover(filePath)
		}
		// Standard fb2
		if ext == ".fb2" {
			return getFb2Cover(filePath)
		}
		// Fallback for fb2.zip if listed as such
		return getFb2Cover(filePath)
	default:
		return nil, "", errors.New("unsupported file type")
	}
}
