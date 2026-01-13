package cover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCover(t *testing.T) {
	// Locate books dir relative to this test file (internal/cover)
	// ../../books
	wd, _ := os.Getwd()
	booksDir := filepath.Join(wd, "..", "..", "books")

	tests := []struct {
		filename string
		wantErr  bool
	}{
		{"test.fb2", false},
		{"nonexistent.epub", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			path := filepath.Join(booksDir, tt.filename)
			data, mimeType, err := GetCover(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCover() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(data) == 0 {
					t.Error("GetCover() returned empty data")
				}
				if mimeType == "" {
					t.Error("GetCover() returned empty mime type")
				}
				t.Logf("Successfully extracted cover from %s: %d bytes, type: %s", tt.filename, len(data), mimeType)
			}
		})
	}
}
