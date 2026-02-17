package scraper

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"olx-hunter/internal/database"
	"olx-hunter/internal/models"
)

type ScraperService struct {
	db       *database.DB
	scraper  *OLXScraper

	activeFilters map[uint]*database.UserFilter
	filtersMutex  sync.RWMutex
}

func NewScraperService(db *database.DB) *ScraperService {
	return &ScraperService{
		db:            db,
		scraper:       NewOLXScraper(),
		activeFilters: make(map[uint]*database.UserFilter),
	}
}

func (s *ScraperService) Cleanup() {
	log.Println("Cleanup completed")
}

func (s *ScraperService) LoadExistingFilters() error {
	log.Println("Load existing filters from database...")

	filters, err := s.db.GetActiveFilters()
	if err != nil {
		return err
	}

	s.filtersMutex.Lock()
	defer s.filtersMutex.Unlock()

	for _, filter := range filters {
		s.activeFilters[filter.ID] = filter
		log.Printf("Loaded filter: ID=%d, Query='%s', UserID=%d",
			filter.ID, filter.Query, filter.UserID)
	}

	log.Printf("Loaded %d active filters for monitoring", len(filters))
	return nil
}

func (s *ScraperService) StartPeriodicScraping(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Wating 30 seconds before first scraping...")
	time.Sleep(30 * time.Second)

	log.Println("Starting initial scraping...")
	s.scrapeAllFilters()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping periodic scraper due to shutdown signal...")
			return
		case <-ticker.C:
			log.Println("Starting scheduled scraping session...")
			s.scrapeAllFilters()
		}
	}
}

func (s *ScraperService) scrapeAllFilters() {
	startTime := time.Now()

	s.filtersMutex.RLock()

	filters := make([]*database.UserFilter, 0, len(s.activeFilters))
	for _, filter := range s.activeFilters {
		filters = append(filters, filter)
	}

	s.filtersMutex.RUnlock()

	if len(filters) == 0 {
		log.Println("No active filters to scrape")
		return
	}

	log.Printf("Starting scraping session: %d filters to process", len(filters))

	successCount := 0
	errorCount := 0

	for i, filter := range filters {
		log.Printf("[%d/%d] Processing filter: ID=%d, Query='%s'",
			i+1, len(filters), filter.ID, filter.Query)

		if err := s.scrapeFilter(filter); err != nil {
			log.Printf("[%d/%d] Error scraping filter %d: %v",
				i+1, len(filters), filter.ID, err)
			errorCount++
		} else {
			log.Printf("✅ [%d/%d] Successfully scraped filter %d",
				i+1, len(filters), filter.ID)
			successCount++
		}

		if i < len(filters)-1 {
			time.Sleep(3 * time.Second)
		}
	}

	duration := time.Since(startTime)
	log.Printf("Scraping session completed:")
	log.Printf("    Duration: %v", duration.Round(time.Second))
	log.Printf("    Success: %d filters", successCount)
	log.Printf("    Errors: %d filters", errorCount)
	log.Printf("    Total: %d filters", len(filters))
}

func (s *ScraperService) scrapeFilter(filter *database.UserFilter) error {
	log.Printf("Scraping filter: ID=%d, Query='%s'", filter.ID, filter.Query)

	searchFilters := models.SearchFilters{
		Query:    filter.Query,
		MinPrice: filter.MinPrice,
		MaxPrice: filter.MaxPrice,
		City:     filter.City,
	}

	listings, err := s.scraper.SearchListings(searchFilters)
	if err != nil {
		return fmt.Errorf("failed to scrape OLX: %w", err)
	}

	log.Printf("Found %d listings for filter %d", len(listings), filter.ID)

	if len(listings) == 0 {
		return nil
	}

	existingURLs, err := s.db.GetExistingURLs(filter.ID)
	if err != nil {
		log.Printf("Failed to get existing URLs: %v", err)
		existingURLs = []string{}
	}

	existingMap := make(map[string]bool)
	for _, url := range existingURLs {
		existingMap[url] = true
	}

	var newListings []models.Listing
	for _, listing := range listings {
		if !existingMap[listing.URL] {
			newListings = append(newListings, listing)

			if err := s.db.SaveListing(filter.ID, listing); err != nil {
				log.Printf("Failed to save listing %s: %v", listing.URL, err)
			}
		}
	}

	var notifiableListings []models.Listing
	for _, listing := range newListings {
		isNotified, err := s.db.IsListingNotified(listing.URL)
		if err != nil {
			log.Printf("Error checking is_notified for %s: %v", listing.URL, err)
		}
		if !isNotified {
			notifiableListings = append(notifiableListings, listing)
		}
	}

	log.Printf("📈 Statistics for filter %d:", filter.ID)
	log.Printf("    Total found: %d", len(listings))
	log.Printf("    Already known: %d", len(listings)-len(newListings))
	log.Printf("    New listings: %d", len(newListings))
	log.Printf("    Already notified: %d", len(newListings)-len(notifiableListings))
	log.Printf("    Ready to notify: %d", len(notifiableListings))

	return nil
}
