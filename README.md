# LegiTrack - Legal Compliance Web Scraper

LegiTrack is a Go-based web scraper designed to monitor legal compliance updates from various government and regulatory websites. It automatically scrapes configured websites at specified intervals and stores the updates in a SQLite database.

## Features

- **Configurable Sources**: Monitor multiple legal compliance websites through a YAML configuration file
- **Automatic Scheduling**: Uses cron expressions to schedule scraping at specified intervals
- **Duplicate Detection**: Prevents storing duplicate content using SHA-256 hashing
- **SQLite Storage**: Stores all scraped data in a local SQLite database
- **Graceful Shutdown**: Handles shutdown signals properly
- **Error Handling**: Comprehensive error handling and logging
- **User Agent Customization**: Configurable user agent string

## Quick Start

### Prerequisites

- Go 1.24.2 or later
- SQLite3

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd LegiTrack
```

2. Install dependencies:
```bash
go mod tidy
```

3. Configure your sources in `config.yaml` (see Configuration section below)

4. Run the scraper:
```bash
go run .
```

For development mode (includes test sources):
```bash
LEGITRACK_ENV=development go run .
```

## Configuration

The scraper uses a YAML configuration file (`config.yaml`) to define:

### Sources

Each source represents a website to monitor:

```yaml
sources:
  indian_gazette:
    id: "indian_gazette"
    name: "Indian Government Gazette"
    url: "https://egazette.gov.in/"
    description: "Official legal notifications and government orders"
    cron: "0 */2 * * *"  # Every 2 hours
    js_rendered: false
    max_retries: 5
    timeout: 60s
    category: "government_official"
```

### Configuration Options

- **id**: Unique identifier for the source
- **name**: Human-readable name
- **url**: Website URL to scrape
- **description**: Description of the source
- **cron**: Cron expression for scheduling (supports seconds for testing)
- **js_rendered**: Whether the site requires JavaScript rendering
- **max_retries**: Maximum number of retry attempts
- **timeout**: Request timeout
- **category**: Category for organizing sources

### Global Settings

```yaml
global:
  default_timeout: 30s
  default_max_retries: 3
  user_agent: "LegiTrack-Bot/1.0 (Legal Compliance Monitor)"
```

## Database Schema

The scraper creates a SQLite database with the following table:

```sql
CREATE TABLE updates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id TEXT NOT NULL,
    url TEXT NOT NULL,
    fetched_at TIMESTAMP NOT NULL,
    hash TEXT UNIQUE,
    status_code INTEGER NOT NULL,
    success BOOLEAN NOT NULL,
    retry_count INTEGER NOT NULL,
    error_detail TEXT,
    body_size INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Usage Examples

### Basic Usage

```bash
# Run with default config.yaml
go run .

# Run with custom config file
go run . custom-config.yaml
```

### Development Mode

```bash
# Enable test sources for development
LEGITRACK_ENV=development go run .
```

## Monitoring Legal Compliance Websites

The scraper is pre-configured to monitor several important legal compliance websites:

1. **Indian Government Gazette** - Official legal notifications
2. **Supreme Court of India** - Latest judgments
3. **Ministry of Law and Justice** - Legal policy updates
4. **RBI Banking Regulations** - Banking sector updates
5. **SEBI Updates** - Securities market regulations

## Adding New Sources

To add a new legal compliance website:

1. Edit `config.yaml`
2. Add a new entry under `sources`:
```yaml
your_source_name:
  id: "unique_id"
  name: "Source Name"
  url: "https://example.com"
  description: "Description of the source"
  cron: "0 */6 * * *"  # Every 6 hours
  js_rendered: false
  max_retries: 3
  timeout: 30s
  category: "your_category"
```

3. Restart the scraper

## Logging

The scraper provides detailed logging with different prefixes:
- `[ORCHESTRATOR]`: Main orchestration and scheduling
- `[HTTP]`: HTTP scraping operations
- `[BROWSER]`: Browser-based scraping (future)
- `[STORAGE]`: Database operations

## Future Enhancements

- Browser-based scraping for JavaScript-heavy sites
- Email notifications for new updates
- Webhook notifications
- Content parsing and filtering
- API endpoints for querying stored data
- Backup and retention policies

## License

[Add your license information here]

## Contributing

[Add contribution guidelines here] 