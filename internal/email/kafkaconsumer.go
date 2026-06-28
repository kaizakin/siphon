package email

import (
	"context"
	"encoding/json"

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
      Topic: topic,
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

    err = json.Unmarshal(msg.Value, &event)
    if err != nil {
      continue
    }

    err = c.router.Handle(ctx, event)
    if err != nil {
      // TODO: retry logic
      // publish to DLQ
      continue
    }
  }
}
