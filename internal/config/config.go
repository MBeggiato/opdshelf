package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	Host             string
	BooksDir         string
	ReverseProxy     bool
	ReverseProxyHost string
	ReverseProxyPort string
}

func LoadConfig() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Determine executable directory
	exePath, err := os.Executable()
	defaultBooksDir := "./books"
	if err == nil {
		defaultBooksDir = filepath.Join(filepath.Dir(exePath), "books")
	} else {
		log.Printf("Error getting executable path: %v", err)
	}

	return &Config{
		Port:             getEnv("PORT", "3000"),
		Host:             getEnv("HOST", "0.0.0.0"),
		BooksDir:         getEnv("BOOKS_DIR", defaultBooksDir),
		ReverseProxy:     strings.ToLower(getEnv("REVERSE_PROXY", "false")) == "true",
		ReverseProxyHost: getEnv("REVERSE_PROXY_HOST", "0.0.0.0"),
		ReverseProxyPort: getEnv("REVERSE_PROXY_PORT", "80"),
	}
}

// Helper function to read environment variables with a fallback
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
