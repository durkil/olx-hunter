package kafka

import (
	"context"
    "encoding/json"
    "log"
    "time"

    "github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic: "olx-events",
		GroupID: groupID,
		StartOffset: kafka.LastOffset,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait: 1 * time.Second,
	})

	return &Consumer{reader: reader}
}

func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.ReadMessage(ctx)
}

func (c *Consumer) ProcessEvents(ctx context.Context, handler EventHandler) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer stopping...")
			return ctx.Err()
		default:
			message, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}
			
			if err := c.handleMessage(message, handler); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

type EventHandler interface {
	HandleFilterCreated(event FilterCreatedEvent) error
	HandleScrapeRequest(event ScrapeRequestEvent) error
	HandleNewListings(event NewListingsEvent) error
}

func (c *Consumer) handleMessage(message kafka.Message, handler EventHandler) error {
	log.Printf("Received message: key=%s, partition=%d, offset=%d",
		string(message.Key), message.Partition, message.Offset)

	var eventType string
	var eventData map[string]interface{}

	if err := json.Unmarshal(message.Value, &eventData); err != nil {
		return err
	}

	eventType, ok := eventData["event_type"].(string)
	if !ok {
		log.Println("Unknown event format")
		return nil
	}

	switch eventType {
	case EventFilterCreated:
		var event FilterCreatedEvent
		if err := json.Unmarshal(message.Value, &event); err != nil {
			return err
		}
		return handler.HandleFilterCreated(event)

	case EventScrapeRequest:
		var event ScrapeRequestEvent
		if err := json.Unmarshal(message.Value, &event); err != nil {
			return err
		}
		return handler.HandleScrapeRequest(event)
	
	case EventNewListings:
		var event NewListingsEvent
		if err := json.Unmarshal(message.Value, &event); err != nil {
			return err
		}
		return handler.HandleNewListings(event)

	default:
		log.Printf("Unknown event type: %s", eventType)
		return nil
	}	
}