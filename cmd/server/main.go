package main

import (
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"opds-server/internal/config"
	"opds-server/internal/handlers"
	"opds-server/internal/middleware"
	"opds-server/internal/utils"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Ensure books directory exists
	if err := os.MkdirAll(cfg.BooksDir, 0755); err != nil {
		log.Fatalf("Failed to create books directory: %v", err)
	}

	// Register mime types if needed
	mime.AddExtensionType(".epub", "application/epub+zip")
	mime.AddExtensionType(".fb2", "application/x-fictionbook+xml")
	mime.AddExtensionType(".fb2.zip", "application/zip")

	// Register template functions globally if needed, though handlers handle their own templates.
	// However, we might want to ensure they are consistent.
	// The original main.go did:
	// templates := template.New("")
	// templates.Funcs(...)
	// But it didn't use this variable 'templates' anywhere except to show it can be done?
	// Actually, the specific handlers (opdsIndexHandler, adminHandler) created their own templates.
	// So we don't need a global template instance here.

	// Helper for checking template functions availability (optional)
	_ = template.FuncMap{
		"formatSize": utils.FormatSize,
		"formatDate": utils.FormatDate,
	}

	// Initialize handlers
	h := handlers.NewHandler(cfg)

	// Create router
	r := mux.NewRouter()

	// Middleware for logging requests
	r.Use(middleware.LoggingMiddleware)

	// Routes
	r.HandleFunc("/", h.OpdsIndexHandler).Methods("GET")
	r.HandleFunc("/admin", h.AdminHandler).Methods("GET")
	r.HandleFunc("/upload", h.UploadHandler).Methods("POST")
	r.HandleFunc("/delete/{filename:.+}", h.DeleteHandler).Methods("POST")
	r.HandleFunc("/simple", h.SimpleHandler).Methods("GET")
	r.HandleFunc("/cover/{filename:.+}", h.CoverHandler).Methods("GET")
	r.HandleFunc("/rename", h.RenameHandler).Methods("POST")

	// Serve books directory
	r.PathPrefix("/books/").Handler(http.StripPrefix("/books/", http.FileServer(http.Dir(cfg.BooksDir))))

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Start server
	serverAddr := cfg.Host + ":" + cfg.Port
	fmt.Printf("OPDS Server running at http://%s\n", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, r))
}
