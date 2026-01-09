package models

import "time"

// BookInfo represents metadata about a book
type BookInfo struct {
	Filename    string
	Title       string
	MimeType    string
	LastUpdated time.Time
	Size        int64
}

// TemplateData holds data passed to HTML templates
type TemplateData struct {
	Books       []BookInfo
	BaseURL     string
	CurrentTime string
}
