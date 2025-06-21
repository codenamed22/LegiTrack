package main

import "time"

// Source defines a single authoritative source to scrape
type Source struct {
	ID         string
	URL        string
	Cron       string
	JSRendered bool
	MaxRetries int
	Timeout    time.Duration
}

// Update represents a scraped update
type Update struct {
	SourceID    string
	URL         string
	FetchedAt   time.Time
	Hash        string
	Body        []byte
	StatusCode  int
	Success     bool
	RetryCount  int
	ErrorDetail string
	Title       string
	Summary     string
	ContentType string
}
