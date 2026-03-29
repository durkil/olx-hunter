package scraper

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"olx-hunter/internal/database"
	"olx-hunter/internal/models"
)

type ScraperService struct {
	db             *database.DB
	scraper        *OLXScraper
	notifyCh       chan<- models.Notification
	workerCount    int
	scrapeInterval time.Duration

	activeFilters map[uint]*database.UserFilter
	filtersMutex  sync.RWMutex
}

func NewScraperService(db *database.DB, notifyCh chan<- models.Notification, workerCount, scrapeIntervalSec int) *ScraperService {
	if workerCount < 1 {
		workerCount = 3
	}
	if scrapeIntervalSec < 30 {
		scrapeIntervalSec = 60
	}
	return &ScraperService{
		db:             db,
		scraper:        NewOLXScraper(),
		notifyCh:       notifyCh,
		workerCount:    workerCount,
		scrapeInterval: time.Duration(scrapeIntervalSec) * time.Second,
		activeFilters:  make(map[uint]*database.UserFilter),
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

func (s *ScraperService) AddFilter(filter *database.UserFilter) {
	s.filtersMutex.Lock()
	defer s.filtersMutex.Unlock()
	s.activeFilters[filter.ID] = filter
	log.Printf("Filter added to scraper: ID=%d, Query='%s'", filter.ID, filter.Query)
}

func (s *ScraperService) RemoveFilter(filterID uint) {
	s.filtersMutex.Lock()
	defer s.filtersMutex.Unlock()
	delete(s.activeFilters, filterID)
	log.Printf("Filter removed from scraper: ID=%d", filterID)
}

func (s *ScraperService) StartPeriodicScraping(ctx context.Context) {
	ticker := time.NewTicker(s.scrapeInterval)
	defer ticker.Stop()

	log.Println("Waiting 30 seconds before first scraping...")
	select {
	case <-time.After(30 * time.Second):
	case <-ctx.Done():
		log.Println("Shutdown during initial delay")
		return
	}

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

	log.Printf("Starting scraping session: %d filters, %d workers", len(filters), s.workerCount)

	var successCount, errorCount int64
	jobs := make(chan *database.UserFilter, len(filters))
	var wg sync.WaitGroup

	for w := 0; w < s.workerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for filter := range jobs {
				log.Printf("[worker %d] Processing filter ID=%d, Query='%s'",
					workerID, filter.ID, filter.Query)

				if err := s.scrapeFilter(filter); err != nil {
					log.Printf("[worker %d] Error filter %d: %v", workerID, filter.ID, err)
					atomic.AddInt64(&errorCount, 1)
				} else {
					log.Printf("[worker %d] ✅ Done filter %d", workerID, filter.ID)
					atomic.AddInt64(&successCount, 1)
				}

				time.Sleep(2 * time.Second)
			}
		}(w)
	}

	for _, filter := range filters {
		jobs <- filter
	}
	close(jobs)

	wg.Wait()

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

	isFirstScrape := len(existingURLs) == 0

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

	if isFirstScrape {
		for _, listing := range newListings {
			if err := s.db.MarkListingAsNotified(listing.URL); err != nil {
				log.Printf("Failed to mark baseline listing %s: %v", listing.URL, err)
			}
		}
		log.Printf("📸 First scrape for filter %d: saved %d listings as baseline (no notification)",
			filter.ID, len(newListings))
		return nil
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
	log.Printf("    Ready to notify: %d", len(notifiableListings))

	if len(notifiableListings) > 0 {
		s.notifyCh <- models.Notification{
			TelegramID: filter.User.TelegramID,
			FilterName: filter.Name,
			Listings:   notifiableListings,
		}

		for _, listing := range notifiableListings {
			if err := s.db.MarkListingAsNotified(listing.URL); err != nil {
				log.Printf("Failed to mark listing as notified %s: %v", listing.URL, err)
			}
		}
	}

	return nil
}
