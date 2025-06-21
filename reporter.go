package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReportData contains all data needed for report generation
type ReportData struct {
	Date        string
	DailyStats  map[string]interface{}
	SourceStats map[string]map[string]interface{}
	Updates     []Update
	Sources     map[string]SourceConfig
}

// Reporter handles HTML report generation
type Reporter struct {
	storage   Storage
	config    *Config
	outputDir string
}

// NewReporter creates a new reporter instance
func NewReporter(storage Storage, config *Config, outputDir string) *Reporter {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("[REPORTER] Failed to create output directory: %v", err)
	}

	return &Reporter{
		storage:   storage,
		config:    config,
		outputDir: outputDir,
	}
}

// GenerateDailyReport generates an HTML report for a specific date
func (r *Reporter) GenerateDailyReport(ctx context.Context, date time.Time) error {
	log.Printf("[REPORTER] Generating daily report for %s", date.Format("2006-01-02"))

	// Get daily statistics
	dailyStats, err := r.storage.GetDailyStats(ctx, date)
	if err != nil {
		return fmt.Errorf("failed to get daily stats: %w", err)
	}

	// Get source statistics
	sourceStats, err := r.storage.GetSourceStats(ctx, date)
	if err != nil {
		return fmt.Errorf("failed to get source stats: %w", err)
	}

	// Get all updates for the date
	updates, err := r.storage.GetUpdatesByDateRange(ctx, date, date)
	if err != nil {
		return fmt.Errorf("failed to get updates: %w", err)
	}

	// Create report data
	reportData := ReportData{
		Date:        date.Format("2006-01-02"),
		DailyStats:  dailyStats,
		SourceStats: sourceStats,
		Updates:     updates,
		Sources:     r.config.Sources,
	}

	// Generate HTML report
	htmlContent, err := r.generateHTML(reportData)
	if err != nil {
		return fmt.Errorf("failed to generate HTML: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("report_%s.html", date.Format("2006-01-02"))
	filepath := filepath.Join(r.outputDir, filename)

	if err := os.WriteFile(filepath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	log.Printf("[REPORTER] Daily report generated: %s", filepath)
	return nil
}

// generateHTML generates the HTML content for the report
func (r *Reporter) generateHTML(data ReportData) (string, error) {
	const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LegiTrack Daily Report - {{.Date}}</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 2.5em;
            font-weight: 300;
        }
        .header .date {
            font-size: 1.2em;
            opacity: 0.9;
            margin-top: 10px;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 30px;
            background: #f8f9fa;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        .stat-number {
            font-size: 2em;
            font-weight: bold;
            color: #667eea;
        }
        .stat-label {
            color: #666;
            margin-top: 5px;
        }
        .content {
            padding: 30px;
        }
        .section {
            margin-bottom: 40px;
        }
        .section h2 {
            color: #333;
            border-bottom: 2px solid #667eea;
            padding-bottom: 10px;
            margin-bottom: 20px;
        }
        .source-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }
        .source-card {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #667eea;
        }
        .source-name {
            font-weight: bold;
            color: #333;
            margin-bottom: 10px;
        }
        .updates-list {
            list-style: none;
            padding: 0;
        }
        .update-item {
            background: white;
            margin-bottom: 15px;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #28a745;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        .update-item.error {
            border-left-color: #dc3545;
        }
        .update-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        .update-source {
            font-weight: bold;
            color: #667eea;
        }
        .update-time {
            color: #666;
            font-size: 0.9em;
        }
        .update-title {
            font-size: 1.1em;
            margin-bottom: 10px;
            color: #333;
        }
        .update-summary {
            color: #666;
            margin-bottom: 10px;
        }
        .update-link {
            color: #667eea;
            text-decoration: none;
        }
        .update-link:hover {
            text-decoration: underline;
        }
        .error-detail {
            color: #dc3545;
            font-style: italic;
        }
        .footer {
            background: #f8f9fa;
            padding: 20px;
            text-align: center;
            color: #666;
            border-top: 1px solid #dee2e6;
        }
        @media (max-width: 768px) {
            .stats-grid {
                grid-template-columns: 1fr;
            }
            .source-stats {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>LegiTrack Daily Report</h1>
            <div class="date">{{.Date}}</div>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-number">{{.DailyStats.total_updates}}</div>
                <div class="stat-label">Total Updates</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">{{.DailyStats.successful_updates}}</div>
                <div class="stat-label">Successful</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">{{.DailyStats.failed_updates}}</div>
                <div class="stat-label">Failed</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">{{.DailyStats.unique_sources}}</div>
                <div class="stat-label">Sources</div>
            </div>
        </div>

        <div class="content">
            <div class="section">
                <h2>Source Statistics</h2>
                <div class="source-stats">
                    {{range $sourceID, $stats := .SourceStats}}
                    <div class="source-card">
                        <div class="source-name">{{$sourceID}}</div>
                        <div>Total: {{$stats.total_updates}}</div>
                        <div>Successful: {{$stats.successful_updates}}</div>
                        <div>Failed: {{$stats.failed_updates}}</div>
                    </div>
                    {{end}}
                </div>
            </div>

            <div class="section">
                <h2>Updates</h2>
                {{if .Updates}}
                <ul class="updates-list">
                    {{range .Updates}}
                    <li class="update-item {{if not .Success}}error{{end}}">
                        <div class="update-header">
                            <span class="update-source">{{.SourceID}}</span>
                            <span class="update-time">{{.FetchedAt.Format "15:04:05"}}</span>
                        </div>
                        {{if .Title}}
                        <div class="update-title">{{.Title}}</div>
                        {{end}}
                        {{if .Summary}}
                        <div class="update-summary">{{.Summary}}</div>
                        {{end}}
                        <div>
                            <a href="{{.URL}}" class="update-link" target="_blank">View Original</a>
                            {{if .Hash}}
                            <span style="margin-left: 10px; color: #666;">Hash: {{.Hash}}</span>
                            {{end}}
                        </div>
                        {{if not .Success}}
                        <div class="error-detail">Error: {{.ErrorDetail}}</div>
                        {{end}}
                    </li>
                    {{end}}
                </ul>
                {{else}}
                <p>No updates found for this date.</p>
                {{end}}
            </div>
        </div>

        <div class="footer">
            <p>Generated by LegiTrack on {{.Date}} at {{now.Format "15:04:05"}}</p>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"now": time.Now,
	}).Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GenerateIndexReport generates an index page with links to all daily reports
func (r *Reporter) GenerateIndexReport(ctx context.Context) error {
	// This would list all available reports
	// For now, we'll create a simple index
	const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LegiTrack Reports Index</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 30px;
        }
        h1 {
            color: #333;
            text-align: center;
            margin-bottom: 30px;
        }
        .report-link {
            display: block;
            padding: 15px;
            margin: 10px 0;
            background: #f8f9fa;
            border-radius: 5px;
            text-decoration: none;
            color: #667eea;
            border-left: 4px solid #667eea;
        }
        .report-link:hover {
            background: #e9ecef;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>LegiTrack Reports</h1>
        <p>Select a date to view the daily report:</p>
        <!-- Reports will be listed here -->
    </div>
</body>
</html>`

	filepath := filepath.Join(r.outputDir, "index.html")
	return os.WriteFile(filepath, []byte(indexTemplate), 0644)
}
