package scraper

import(
	"net/http"
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
	return []Listing{
		{URL: "https://www.olx.ua/d/uk/obyavlenie/iphone-15-test-1.html"},
		{URL: "https://www.olx.ua/d/uk/obyavlenie/iphone-15-test-2.html"},
	}, nil
}