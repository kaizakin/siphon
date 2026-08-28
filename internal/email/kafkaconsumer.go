package email

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	router *Router
}

func NewConsumer(brokers []string, topic string, groupID string, router *Router) *Consumer {
	reader := kafka.NewReader(
		kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		},
	)

	return &Consumer{
		reader: reader,
		router: router,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}

		var event Event

		log.Printf("[%s] processing event %s for recipient=%s user_id=%s source=%s", event.CorrelationID, event.EventID, event.Recipient, event.Metadata["user_id"], event.Source)

		err = json.Unmarshal(msg.Value, &event)
		if err != nil {
			log.Printf("failed to unmarshal kafka message: %v", err)
			continue
		}

		err = c.router.Handle(ctx, event)
		if err != nil {
			log.Printf("failed to process event %q (id=%s): %v", event.EventType, event.EventID, err)
			// TODO: retry logic / publish to DLQ
			continue
		}
	}
}
