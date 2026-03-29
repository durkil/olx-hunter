# OLX Hunter

Telegram bot for real-time monitoring of OLX listings. Create search filters with keywords, price range and city — the bot automatically scrapes OLX and sends notifications about new listings.

## Features

- **Step-by-step filter creation** via Telegram bot
- **Real-time scraping** with configurable interval and worker pool
- **Smart notifications** — inline "Show" button, no spam, old messages auto-deleted
- **Baseline mechanism** — first scrape saves existing listings without notification, only truly new ones trigger alerts
- **Redis caching** with rate limiting to prevent IP bans
- **Filter management** — create, delete, enable/disable filters on the fly
- **Graceful shutdown** with context cancellation

## Tech Stack

- **Go 1.24**
- **PostgreSQL** — users, filters, saved listings
- **Redis** — search result caching, rate limiting
- **Colly** — web scraping
- **Telegram Bot API** — user interface
- **Docker Compose** — local infrastructure

## Project Structure

```
├── cmd/
│   └── main.go                  # Entry point
├── internal/
│   ├── bot/bot.go               # Telegram bot, commands, callbacks
│   ├── scraper/
│   │   ├── scraper.go           # OLX scraper (Colly)
│   │   └── service.go           # Periodic scraping with worker pool
│   ├── database/
│   │   ├── models.go            # GORM models
│   │   └── crud.go              # Database operations
│   ├── cache/redis.go           # Redis client
│   ├── config/config.go         # Environment config
│   ├── models/listing.go        # Shared models
│   └── utils/time_converter.go  # Time utilities
├── migrations/                  # SQL migrations
├── docker-compose.yml
└── .env
```

## Setup

### 1. Clone and configure

```bash
git clone https://github.com/yourusername/olx-hunter.git
cd olx-hunter
cp .env.example .env
```

Edit `.env` with your values:

```env
BOT_TOKEN=your_telegram_bot_token

DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=olx_hunter
DB_PORT=5432
DB_SSLMODE=disable

REDIS_ADDR=localhost:6379

WORKER_COUNT=5
SCRAPE_INTERVAL=60
```

### 2. Start infrastructure

```bash
docker-compose up -d
```

### 3. Run

```bash
go run cmd/main.go
```

## Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Welcome message |
| `/help` | Show all commands |
| `/create` | Create new filter (step-by-step) |
| `/list` | Show your filters |
| `/find [num]` | Search listings by filter |
| `/toggle [num]` | Enable/disable filter |
| `/delete [num]` | Delete filter |

## How It Works

```
User creates filter ──> Scraper picks it up immediately
                              │
                    Scrapes OLX every N seconds
                              │
                    Compares with saved listings
                              │
               New listings found? ──> Notification channel
                                              │
                                   Bot sends "🔔 Found X new listings"
                                       with [Show] button
                                              │
                                   User clicks ──> Photos + details
```

## Running Tests

```bash
go test ./internal/database/ -v
```
