package main

import (
	"fmt"
	"olx-hunter/internal/scraper"
)

func main() {
	fmt.Println("OLX scraper is started!")

	s := scraper.NewOLXScraper()
	filters := scraper.SearchFilters{
		Query: "iphone-15-pro-max",
		MinPrice: 25000,
		MaxPrice: 30000,
		City: "одеса",
	}
	listings, err := s.SearchListings(filters)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d listings:\n", len(listings))
	for i, listing := range listings {
		fmt.Printf("%d. %s (%s) - %s\n%s\n\n", 
		i+1,
		listing.Title, 
		listing.Price,
		listing.Location,
		listing.URL)
	}
}