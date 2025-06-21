package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Global        GlobalConfig            `yaml:"global"`
	Sources       map[string]SourceConfig `yaml:"sources"`
	TestSources   map[string]SourceConfig `yaml:"test_sources"`
	Notifications NotificationConfig      `yaml:"notifications"`
	Storage       StorageConfig           `yaml:"storage"`
	Reporting     ReportingConfig         `yaml:"reporting"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	DefaultTimeout    string `yaml:"default_timeout"`
	DefaultMaxRetries int    `yaml:"default_max_retries"`
	UserAgent         string `yaml:"user_agent"`
}

// SourceConfig represents a single source configuration
type SourceConfig struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
	Description string `yaml:"description"`
	Cron        string `yaml:"cron"`
	JSRendered  bool   `yaml:"js_rendered"`
	MaxRetries  int    `yaml:"max_retries"`
	Timeout     string `yaml:"timeout"`
	Category    string `yaml:"category"`
}

// NotificationConfig contains notification settings
type NotificationConfig struct {
	Email   EmailConfig   `yaml:"email"`
	Webhook WebhookConfig `yaml:"webhook"`
}

// EmailConfig contains email notification settings
type EmailConfig struct {
	Enabled    bool     `yaml:"enabled"`
	SMTPServer string   `yaml:"smtp_server"`
	SMTPPort   int      `yaml:"smtp_port"`
	Username   string   `yaml:"username"`
	Password   string   `yaml:"password"`
	Recipients []string `yaml:"recipients"`
}

// WebhookConfig contains webhook notification settings
type WebhookConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
	Secret  string `yaml:"secret"`
}

// StorageConfig contains storage settings
type StorageConfig struct {
	DatabasePath     string `yaml:"database_path"`
	BackupEnabled    bool   `yaml:"backup_enabled"`
	BackupInterval   string `yaml:"backup_interval"`
	MaxRetentionDays int    `yaml:"max_retention_days"`
}

// ReportingConfig contains reporting settings
type ReportingConfig struct {
	Enabled          bool   `yaml:"enabled"`
	OutputDirectory  string `yaml:"output_directory"`
	DailyReportTime  string `yaml:"daily_report_time"`
	AutoGenerate     bool   `yaml:"auto_generate"`
	IncludeSummaries bool   `yaml:"include_summaries"`
	IncludeErrors    bool   `yaml:"include_errors"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetSources returns all sources as a slice of Source structs
func (c *Config) GetSources() []Source {
	var sources []Source

	// Add regular sources
	for _, srcConfig := range c.Sources {
		source := c.convertToSource(srcConfig)
		sources = append(sources, source)
	}

	// Add test sources if in development mode
	if os.Getenv("LEGITRACK_ENV") == "development" {
		for _, srcConfig := range c.TestSources {
			source := c.convertToSource(srcConfig)
			sources = append(sources, source)
		}
	}

	return sources
}

// convertToSource converts a SourceConfig to a Source struct
func (c *Config) convertToSource(srcConfig SourceConfig) Source {
	timeout, err := time.ParseDuration(srcConfig.Timeout)
	if err != nil {
		// Fallback to global default
		timeout, _ = time.ParseDuration(c.Global.DefaultTimeout)
	}

	maxRetries := srcConfig.MaxRetries
	if maxRetries == 0 {
		maxRetries = c.Global.DefaultMaxRetries
	}

	return Source{
		ID:         srcConfig.ID,
		URL:        srcConfig.URL,
		Cron:       srcConfig.Cron,
		JSRendered: srcConfig.JSRendered,
		MaxRetries: maxRetries,
		Timeout:    timeout,
	}
}

// GetDatabasePath returns the database path from config
func (c *Config) GetDatabasePath() string {
	if c.Storage.DatabasePath != "" {
		return c.Storage.DatabasePath
	}
	return "./legitrack.db" // Default fallback
}

// GetUserAgent returns the user agent string
func (c *Config) GetUserAgent() string {
	if c.Global.UserAgent != "" {
		return c.Global.UserAgent
	}
	return "LegiTrack-Bot/1.0 (Legal Compliance Monitor)"
}

// GetReportingOutputDir returns the reporting output directory
func (c *Config) GetReportingOutputDir() string {
	if c.Reporting.OutputDirectory != "" {
		return c.Reporting.OutputDirectory
	}
	return "./reports" // Default fallback
}
