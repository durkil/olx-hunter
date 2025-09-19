package main

import (
	"fmt"
	"olx-hunter/internal/scraper"
)

func main() {
	fmt.Println("OLX scraper is started!")

	s := scraper.NewOLXScraper()
	listings, err := s.SearchListings("iPhone")

	if err != nil {
		fmt.Println("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d listings:\n", len(listings))
	for i, listing := range listings {
		fmt.Printf("%d. %s\n", i+1, listing.URL)
	}
}