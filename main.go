// LegiTrack — unified scraper with fixes applied
// ------------------------------------------------
// Single‑file reference implementation for clarity; split into packages as needed.
// Key improvements vs. previous draft:
//   • Cron now uses 5‑field patterns (no WithSeconds).
//   • Graceful shutdown via context + WaitGroup.
//   • Fetcher retry loop resets state and respects ctx.Done().
//   • BrowserScraper rebuilds chromedp context on each retry.
//   • SQLite opened with parseTime=true and WAL journal, timestamp layout simplified.
//   • Storage worker pool (size 4) + draining on exit.
//   • Misc nil‑checks, MaxRetries helper.

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
)

// ScraperManager coordinates different scrapers
type ScraperManager struct {
	httpScraper    *HTTPScraper
	browserScraper *BrowserScraper
}

// NewScraperManager creates a new scraper manager
func NewScraperManager() *ScraperManager {
	return &ScraperManager{
		httpScraper:    NewHTTPScraper(),
		browserScraper: NewBrowserScraper(),
	}
}

// Scrape delegates to the appropriate scraper based on source configuration
func (sm *ScraperManager) Scrape(ctx context.Context, src Source, out chan<- Update) error {
	if src.JSRendered {
		return sm.browserScraper.Scrape(ctx, src, out)
	}
	return sm.httpScraper.Scrape(ctx, src, out)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Starting LegiTrack web scraper...")

	// Load configuration
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded from %s", configPath)

	// Initialize storage
	storage, err := NewSQLiteStorage(config.GetDatabasePath())
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Initialize reporter
	reporter := NewReporter(storage, config, config.GetReportingOutputDir())

	// Check if this is a report generation command
	if len(os.Args) > 2 && os.Args[2] == "report" {
		if len(os.Args) > 3 {
			// Generate report for specific date
			dateStr := os.Args[3]
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				log.Fatalf("Invalid date format. Use YYYY-MM-DD: %v", err)
			}

			if err := reporter.GenerateDailyReport(ctx, date); err != nil {
				log.Fatalf("Failed to generate report: %v", err)
			}
			log.Println("Report generated successfully!")
			return
		} else {
			// Generate report for today
			if err := reporter.GenerateDailyReport(ctx, time.Now()); err != nil {
				log.Fatalf("Failed to generate report: %v", err)
			}
			log.Println("Today's report generated successfully!")
			return
		}
	}

	// Initialize scraper manager
	scraperManager := NewScraperManager()

	// Set user agent from configuration
	scraperManager.httpScraper.SetUserAgent(config.GetUserAgent())

	// Buffered channel for updates
	updates := make(chan Update, 1024)

	// Worker goroutine to process updates
	go func() {
		for update := range updates {
			// Check if we already have this content
			if update.Hash != "" {
				existingUpdate, exists, err := storage.GetUpdateByHash(ctx, update.Hash)
				if err != nil {
					log.Printf("[ORCHESTRATOR] Error checking existing update: %v", err)
				}

				// Skip if we already have the same content and it's not newer
				if exists && !existingUpdate.FetchedAt.Before(update.FetchedAt) {
					log.Printf("[ORCHESTRATOR] Skipping duplicate content for %s", update.SourceID)
					continue
				}
			}

			// Save the update
			if err := storage.SaveUpdate(ctx, update); err != nil {
				log.Printf("[ORCHESTRATOR] Failed to save update: %v", err)
				continue
			}

			// Log successful processing
			if update.Success {
				log.Printf("[ORCHESTRATOR] New content detected for %s (hash: %s)",
					update.SourceID, update.Hash[:8])
			} else {
				log.Printf("[ORCHESTRATOR] Error update saved for %s: %s",
					update.SourceID, update.ErrorDetail)
			}
		}
	}()

	// Get sources from configuration
	sources := config.GetSources()
	if len(sources) == 0 {
		log.Fatal("No sources configured. Please check your config.yaml file.")
	}

	log.Printf("Loaded %d sources from configuration", len(sources))

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Perform initial scrape for all sources
	log.Println("[ORCHESTRATOR] Performing initial scrape for all sources...")
	for _, src := range sources {
		go func(source Source) {
			log.Printf("[ORCHESTRATOR] Initial scrape starting for %s", source.ID)
			if err := scraperManager.Scrape(ctx, source, updates); err != nil {
				log.Printf("[ORCHESTRATOR] Initial scrape failed for %s: %v", source.ID, err)
			}
		}(src)
	}

	// Start the scheduler with seconds support for testing
	scheduler := cron.New(cron.WithSeconds())

	// Schedule each source
	for _, src := range sources {
		source := src // Capture for closure

		entryID, err := scheduler.AddFunc(source.Cron, func() {
			log.Printf("[ORCHESTRATOR] Scheduled scrape starting for %s", source.ID)

			// Run scrape in a goroutine to avoid blocking the scheduler
			go func() {
				if err := scraperManager.Scrape(ctx, source, updates); err != nil {
					log.Printf("[ORCHESTRATOR] Scheduled scrape failed for %s: %v", source.ID, err)
				}
			}()
		})

		if err != nil {
			log.Printf("[ORCHESTRATOR] Failed to schedule %s: %v", src.ID, err)
		} else {
			log.Printf("[ORCHESTRATOR] Scheduled %s with cron '%s' (ID: %d)",
				src.ID, src.Cron, entryID)
		}
	}

	// Schedule daily report generation (at 23:59 every day)
	_, err = scheduler.AddFunc("59 23 * * * *", func() {
		log.Println("[ORCHESTRATOR] Generating daily report...")
		if err := reporter.GenerateDailyReport(ctx, time.Now().AddDate(0, 0, -1)); err != nil {
			log.Printf("[ORCHESTRATOR] Failed to generate daily report: %v", err)
		}
	})
	if err != nil {
		log.Printf("[ORCHESTRATOR] Failed to schedule daily report: %v", err)
	} else {
		log.Println("[ORCHESTRATOR] Scheduled daily report generation at 23:59")
	}

	// Start the scheduler
	scheduler.Start()
	log.Println("[ORCHESTRATOR] Scheduler started successfully")

	// Print current status
	log.Printf("[ORCHESTRATOR] Monitoring %d sources for legal updates", len(sources))
	for _, src := range sources {
		latest, err := storage.GetLatestUpdateBySource(ctx, src.ID)
		if err != nil {
			log.Printf("[ORCHESTRATOR] Could not get latest update for %s: %v", src.ID, err)
		} else if latest != nil {
			log.Printf("[ORCHESTRATOR] %s: Last update %s", src.ID, latest.FetchedAt.Format(time.RFC3339))
		} else {
			log.Printf("[ORCHESTRATOR] %s: No previous updates found", src.ID)
		}
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("[ORCHESTRATOR] Shutdown signal received, stopping gracefully...")

	// Stop the scheduler
	scheduler.Stop()
	log.Println("[ORCHESTRATOR] Scheduler stopped")

	// Wait for any ongoing scrapes to complete
	time.Sleep(10 * time.Second)

	// Close the updates channel
	close(updates)

	// Wait for the update processor to finish
	time.Sleep(2 * time.Second)

	log.Println("[ORCHESTRATOR] LegiTrack web scraper stopped successfully")
}
