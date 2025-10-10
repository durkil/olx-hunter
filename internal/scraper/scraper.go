package scraper

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

type Listing struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Price    string `json:"price"`
	PriceInt int    `json:"price_int"`
	Location string `json:"location"`
}

type SearchFilters struct {
	Query    string `json:"query"`
	MinPrice int    `json:"min_price"`
	MaxPrice int    `json:"max_price"`
	City     string `json:"city"`
}

type Scraper interface {
	SearchListings(filters SearchFilters) ([]Listing, error)
}

// cleanText очищає текст від CSS, HTML та інших артефактів
func cleanText(text string) string {
	// Видаляємо CSS
	cssRegex := regexp.MustCompile(`\.css-[^;]+;|\.css-[^}]+}`)
	text = cssRegex.ReplaceAllString(text, "")

	// Видаляємо CSS властивості
	propertyRegex := regexp.MustCompile(`[a-zA-Z-]+:\s*[^;]+;`)
	text = propertyRegex.ReplaceAllString(text, "")

	// Видаляємо зайві пробіли та переводи рядків
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

type OLXScraper struct {
	client *http.Client
}

func NewOLXScraper() *OLXScraper {
	return &OLXScraper{
		client: &http.Client{},
	}
}

func parsePrice(priceStr string) int {
	cleanedPrice := strings.ReplaceAll(priceStr, " ", "")
	cleanedPrice = strings.ReplaceAll(cleanedPrice, "грн.", "")

	price, err := strconv.Atoi(cleanedPrice)
	if err != nil {
		return 0
	}
	return price
}

func (s *OLXScraper) SearchListings(filters SearchFilters) ([]Listing, error) {
	searchURL := fmt.Sprintf("https://www.olx.ua/uk/list/q-%s/", filters.Query)

	c := colly.NewCollector()

	urlMap := make(map[string]bool)
	var listings []Listing

	c.OnHTML("a[href*='/d/uk/obyavlenie/']", func(e *colly.HTMLElement) {
		fullURL := "https://www.olx.ua" + e.Attr("href")

		if !urlMap[fullURL] {
			urlMap[fullURL] = true

			card := e.DOM.Closest("[data-cy='l-card']")

			title := card.Find("h4").Text()
			priceText := card.Find("p[data-testid='ad-price']").Text()
			location := card.Find("p[data-testid='location-date']").Text()

			listing := Listing{
				URL:      fullURL,
				Title:    cleanText(title),
				Price:    cleanText(priceText),
				PriceInt: parsePrice(priceText),
				Location: cleanText(location),
			}

			if filters.MinPrice > 0 && listing.PriceInt < filters.MinPrice {
				return
			}
			if filters.MaxPrice > 0 && listing.PriceInt > filters.MaxPrice {
				return
			}

			if filters.City != "" {
				cityLower := strings.ToLower(filters.City)
				locationLower := strings.ToLower(listing.Location)
				if !strings.Contains(locationLower, cityLower) {
					return
				}
			}
			listings = append(listings, listing)
		}
	})

	err := c.Visit(searchURL)
	if err != nil {
		return nil, err
	}

	return listings, nil
}
