package handlers

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"opds-server/internal/config"
	"opds-server/internal/cover"
	"opds-server/internal/models"
	"opds-server/internal/utils"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type Handler struct {
	Config *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{Config: cfg}
}

// Template functions to improve display
var templateFuncs = template.FuncMap{
	"formatDate": utils.FormatDate,
	"formatSize": utils.FormatSize,
	"even": func(i int) bool {
		return i%2 == 0
	},
	"simpleMime": func(mimeType string) string {
		switch mimeType {
		case "application/epub+zip":
			return "EPUB"
		case "application/pdf":
			return "PDF"
		case "application/x-fictionbook+xml", "application/x-zip-compressed-fb2":
			return "FB2"
		case "application/zip", "application/x-zip-compressed":
			return "ZIP"
		case "application/x-cbz", "application/vnd.comicbook+zip":
			return "CBZ"
		case "application/x-cbr":
			return "CBR"
		case "application/x-mobi", "application/x-mobipocket-ebook":
			return "MOBI"
		case "application/vnd.amazon.ebook":
			return "AZW"
		case "image/vnd.djvu":
			return "DJVU"
		case "text/plain":
			return "TXT"
		case "text/rtf", "application/rtf":
			return "RTF"
		case "text/html":
			return "HTML"
		default:
			// Improved heuristic: Check if it contains specific keywords
			lower := strings.ToLower(mimeType)
			if strings.Contains(lower, "azw") {
				return "AZW"
			}
			if strings.Contains(lower, "djvu") {
				return "DJVU"
			}
			// Fallback: truncated
			if len(mimeType) > 12 {
				return mimeType[:10] + "..."
			}
			return mimeType
		}
	},
}

// OpdsIndexHandler handles the OPDS catalog request
func (h *Handler) OpdsIndexHandler(w http.ResponseWriter, r *http.Request) {
	books, err := utils.GetBooksList(h.Config.BooksDir)
	if err != nil {
		http.Error(w, "Failed to read books directory", http.StatusInternalServerError)
		return
	}

	sortMode := r.URL.Query().Get("sort")
	utils.SortBooks(books, sortMode)

	data := models.TemplateData{
		Books:       books,
		BaseURL:     h.GetBaseURL(r),
		CurrentTime: time.Now().UTC().Format(time.RFC3339),
		SortMode:    sortMode,
	}

	// Set content type for OPDS
	w.Header().Set("Content-Type", "application/atom+xml;charset=utf-8;profile=opds-catalog;kind=acquisition")

	// Write XML header directly to avoid template escaping
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`)

	// Use template for rest of OPDS XML
	tmpl, err := template.New("opds.xml").Funcs(templateFuncs).ParseFiles("templates/opds.xml")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Error generating OPDS feed", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error generating OPDS feed", http.StatusInternalServerError)
	}
}

// AdminHandler handles the admin page request
func (h *Handler) AdminHandler(w http.ResponseWriter, r *http.Request) {
	books, err := utils.GetBooksList(h.Config.BooksDir)
	if err != nil {
		http.Error(w, "Failed to read books directory", http.StatusInternalServerError)
		return
	}

	sortMode := r.URL.Query().Get("sort")
	if sortMode == "" {
		sortMode = "date-desc" // Default for admin
	}
	utils.SortBooks(books, sortMode)

	data := models.TemplateData{
		Books:    books,
		BaseURL:  h.GetBaseURL(r),
		SortMode: sortMode,
	}

	// Parse templates with function map
	tmpl, err := template.New("layout.html").Funcs(templateFuncs).ParseFiles(
		"templates/layout.html",
		"templates/admin.html",
	)
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Error rendering admin page", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering admin page", http.StatusInternalServerError)
	}
}

// UploadHandler handles file uploads
func (h *Handler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form with 32MB max memory
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("book")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create destination file
	filePath := filepath.Join(h.Config.BooksDir, header.Filename)
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath // Fallback to relative if abs fails
	}
	log.Printf("Starting upload for file: %s to path: %s", header.Filename, absPath)

	dest, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file %s: %v", filePath, err)
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	// Copy file content
	if _, err := io.Copy(dest, file); err != nil {
		log.Printf("Error copying content to %s: %v", filePath, err)
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully uploaded file: %s", header.Filename)

	// Redirect back to admin page
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// DeleteHandler handles file deletion
func (h *Handler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	// Sanitize filename for safety
	filePath := filepath.Join(h.Config.BooksDir, filepath.Clean(filename))
	log.Printf("Attempting to delete file: %s", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("File not found for deletion: %s", filePath)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Delete file
	if err := os.Remove(filePath); err != nil {
		log.Printf("Error deleting file %s: %v", filePath, err)
		http.Error(w, "Error deleting file", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully deleted file: %s", filename)

	// Redirect back to admin page
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// SimpleHandler handles the simple book list page request
func (h *Handler) SimpleHandler(w http.ResponseWriter, r *http.Request) {
	books, err := utils.GetBooksList(h.Config.BooksDir)
	if err != nil {
		http.Error(w, "Failed to read books directory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, "<html><head><title>Simple Book List</title></head><body>")
	fmt.Fprintln(w, "<h1>Book List</h1>")
	if len(books) == 0 {
		fmt.Fprintln(w, "<p>No books available.</p>")
	} else {
		fmt.Fprintln(w, "<ul>")
		for _, book := range books {
			fmt.Fprintf(w, "<li><b>%s</b> (%s, %s)", book.Title, book.MimeType, utils.FormatSize(book.Size))
			fmt.Fprintf(w, " - <a href='/books/%s'>Original</a>", book.Filename)
			fmt.Fprintln(w, "</li>")
		}
		fmt.Fprintln(w, "</ul>")
	}
	fmt.Fprintln(w, "</body></html>")
}

// Helper function to get base URL
func (h *Handler) GetBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if !h.Config.ReverseProxy {
		return fmt.Sprintf("%s://%s", scheme, r.Host)
	} else {
		return fmt.Sprintf("%s://%s:%s", scheme, h.Config.ReverseProxyHost, h.Config.ReverseProxyPort)
	}
}

// CoverHandler handles the cover image request
func (h *Handler) CoverHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	// Sanitize filename for safety
	cleanFilename := filepath.Clean(filename)
	filePath := filepath.Join(h.Config.BooksDir, cleanFilename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	coverData, mimeType, err := cover.GetCover(filePath)
	if err != nil {
		// Log error but return 404 or default image so client can handle it
		log.Printf("Error extracting cover for %s: %v", filename, err)
		http.Error(w, "Cover not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 1 day
	w.Write(coverData)
}

// RenameHandler handles file renaming
func (h *Handler) RenameHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	oldFilename := r.FormValue("oldFilename")
	newFilename := r.FormValue("newFilename")

	if oldFilename == "" || newFilename == "" {
		http.Error(w, "Missing filenames", http.StatusBadRequest)
		return
	}

	// Clean paths to prevent traversal
	cleanOld := filepath.Clean(oldFilename)
	cleanNew := filepath.Clean(newFilename)

	// Preserve extension if user forgot it
	if filepath.Ext(cleanNew) == "" {
		cleanNew += filepath.Ext(cleanOld)
	}

	oldPath := filepath.Join(h.Config.BooksDir, cleanOld)
	// Use the same directory as the old file
	newPath := filepath.Join(filepath.Dir(oldPath), cleanNew)

	// Check if old file exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		http.Error(w, "Original file not found", http.StatusNotFound)
		return
	}

	// Check if new file already exists to prevent overwrite
	if _, err := os.Stat(newPath); err == nil {
		http.Error(w, "Destination file already exists", http.StatusConflict)
		return
	}

	log.Printf("Renaming %s to %s", oldPath, newPath)
	if err := os.Rename(oldPath, newPath); err != nil {
		log.Printf("Error renaming file: %v", err)
		http.Error(w, "Error renaming file", http.StatusInternalServerError)
		return
	}

	// Redirect back to admin
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
