package kafka

import (
	"olx-hunter/internal/scraper"
	"time"
)

const (
	EventFilterCreated = "filter_created"
	EventScrapeRequest = "scrape_request"
	EventNewListings   = "new_listings"
)

type FilterCreatedEvent struct {
	EventType string    `json:"event_type"`
	UserID    int64     `json:"user_id"`
	FilterID  uint      `json:"filter_id"`
	Query     string    `json:"query"`
	MinPrice  int       `json:"min_price"`
	MaxPrice  int       `json:"max_price"`
	City      string    `json:"city"`
	CreatedAt time.Time `json:"created_at"`
}

type ScrapeRequestEvent struct {
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

type NewListingsEvent struct {
	EventType string            `json:"event_type"`
	FilterID  int               `json:"filter_id"`
	UserID    int64             `json:"user_id"`
	Query     string            `json:"query"`
	Listings  []scraper.Listing `json:"listings"`
	FoundAt   time.Time         `json:"found_at"`
}
