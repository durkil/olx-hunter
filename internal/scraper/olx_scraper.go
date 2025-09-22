package scraper

import (
	"fmt"
	"net/http"
	"github.com/gocolly/colly/v2"
)

type Listing struct {
	URL string `json:"url"`
}

type OLXResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
}

type Scraper interface {
	SearchListings(query string) ([]Listing, error)
}

type OLXScraper struct {
	client *http.Client
}

func NewOLXScraper() *OLXScraper {
	return &OLXScraper{
		client: &http.Client{},
	}
}

func (s *OLXScraper) SearchListings(query string) ([]Listing, error) {
	searchURL := fmt.Sprintf("https://www.olx.ua/uk/list/q-%s/", query)

	c := colly.NewCollector()

	urlMap := make(map[string]bool)
	var listings []Listing

	c.OnHTML("a[href*='/d/uk/obyavlenie/']", func(e *colly.HTMLElement) {
		fullURL := "https://www.olx.ua" + e.Attr("href")

		if !urlMap[fullURL] {
			urlMap[fullURL] = true
			listings = append(listings, Listing{URL: fullURL})
		}
	})

	err := c.Visit(searchURL)
	if err != nil {
		return nil, err
	}

	return listings, nil
}
