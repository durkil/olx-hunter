package kafka

import (
	"context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer() *Producer {
	writer := &kafka.Writer{
		Addr: kafka.TCP("localhost:9092"),
		Topic: "olx-events",
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{writer: writer}
}

func (p *Producer) PublishFilterCreated(event FilterCreatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal filter_created event: %w", err)
	}

	message := kafka.Message{
		Key: []byte(fmt.Sprintf("filter_%d", event.FilterID)),
		Value: data,
		Time: time.Now(),
	}

	err = p.writer.WriteMessages(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to write filter_created message: %w", err)
	}

	log.Printf("Published filter_created event: filter_id=%d, user_id=%d", event.FilterID, event.UserID)
	return nil
}

func (p *Producer) PublishScrapeRequest() error {
	event := ScrapeRequestEvent{
		EventType: EventScrapeRequest,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal scrape_request event: %w", err)
	}

	message := kafka.Message{
		Key: []byte("scrape_request"),
		Value: data,
		Time: time.Now(),
	}

	err = p.writer.WriteMessages(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to write scrape_request message: %w", err)
	}

	log.Printf("Published scrape_request event")
	return nil
}

func (p *Producer) PublishNewListings(event NewListingsEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal new_listings event: %w", err)
	}

	message := kafka.Message{
		Key: []byte(fmt.Sprintf("listings_filter_%d", event.FilterID)),
		Value: data,
		Time: time.Now(),
	}

	err = p.writer.WriteMessages(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to write new_listings message: %w", err)
	}

	log.Printf("Published new_listings event: filter_id=%d, count=%d", event.FilterID, len(event.Listings))
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}