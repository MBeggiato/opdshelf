package handlers

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"opds-server/internal/config"
	"opds-server/internal/models"
	"opds-server/internal/utils"
	"os"
	"path/filepath"
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
}

// OpdsIndexHandler handles the OPDS catalog request
func (h *Handler) OpdsIndexHandler(w http.ResponseWriter, r *http.Request) {
	books, err := utils.GetBooksList(h.Config.BooksDir)
	if err != nil {
		http.Error(w, "Failed to read books directory", http.StatusInternalServerError)
		return
	}

	data := models.TemplateData{
		Books:       books,
		BaseURL:     h.GetBaseURL(r),
		CurrentTime: time.Now().UTC().Format(time.RFC3339),
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

	data := models.TemplateData{
		Books:   books,
		BaseURL: h.GetBaseURL(r),
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
