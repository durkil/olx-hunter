package models

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

type Notification struct {
	TelegramID int64
	FilterName string
	Listings   []Listing
}