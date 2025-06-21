package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Scraper interface defines the contract for different scraping implementations
type Scraper interface {
	Scrape(ctx context.Context, src Source, out chan<- Update) error
}

// HTTPScraper handles regular HTTP scraping
type HTTPScraper struct {
	client    *http.Client
	userAgent string
}

// BrowserScraper handles JavaScript-rendered websites (placeholder for now)
type BrowserScraper struct {
	// Will implement later
}

// NewHTTPScraper creates a new HTTP scraper
func NewHTTPScraper() *HTTPScraper {
	return &HTTPScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "LegiTrack-Bot/1.0 (Legal Compliance Monitor)",
	}
}

// SetUserAgent sets the user agent for the HTTP scraper
func (h *HTTPScraper) SetUserAgent(userAgent string) {
	h.userAgent = userAgent
}

// NewBrowserScraper creates a new browser scraper
func NewBrowserScraper() *BrowserScraper {
	return &BrowserScraper{}
}

// Scrape implements HTTP scraping
func (h *HTTPScraper) Scrape(ctx context.Context, src Source, out chan<- Update) error {
	log.Printf("[HTTP] Starting scrape of %s", src.URL)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", src.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", h.userAgent)

	// Make the request
	resp, err := h.client.Do(req)
	if err != nil {
		// Send error update
		out <- Update{
			SourceID:    src.ID,
			URL:         src.URL,
			FetchedAt:   time.Now().UTC(),
			Success:     false,
			ErrorDetail: err.Error(),
		}
		return fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		out <- Update{
			SourceID:    src.ID,
			URL:         src.URL,
			FetchedAt:   time.Now().UTC(),
			StatusCode:  resp.StatusCode,
			Success:     false,
			ErrorDetail: err.Error(),
		}
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Create hash
	hash := sha256.Sum256(body)

	// Send successful update
	out <- Update{
		SourceID:   src.ID,
		URL:        src.URL,
		FetchedAt:  time.Now().UTC(),
		Hash:       hex.EncodeToString(hash[:]),
		Body:       body,
		StatusCode: resp.StatusCode,
		Success:    true,
		RetryCount: 0,
	}

	log.Printf("[HTTP] Successfully scraped %s (status: %d, size: %d bytes)", src.URL, resp.StatusCode, len(body))
	return nil
}

// Scrape implements browser scraping (placeholder)
func (b *BrowserScraper) Scrape(ctx context.Context, src Source, out chan<- Update) error {
	log.Printf("[BROWSER] Browser scraping not implemented yet for %s", src.URL)

	// For now, just send an error update
	out <- Update{
		SourceID:    src.ID,
		URL:         src.URL,
		FetchedAt:   time.Now().UTC(),
		Success:     false,
		ErrorDetail: "Browser scraping not implemented yet",
	}

	return fmt.Errorf("browser scraping not implemented")
}
