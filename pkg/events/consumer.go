package events

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"github.com/FahadAkash/100M-Systems/pkg/metrics"
	"github.com/segmentio/kafka-go"
)

// ClickBatchHandler is invoked when a micro-batch of ClickEvents is ready to persist.
type ClickBatchHandler func(ctx context.Context, batch []ClickEvent) error

// KafkaConsumer reads click events from a Kafka topic and micro-batches them into persistent storage.
type KafkaConsumer struct {
	reader  *kafka.Reader
	handler ClickBatchHandler
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewKafkaConsumer initializes a consumer reader for a consumer group.
func NewKafkaConsumer(brokers []string, topic, groupID string, handler ClickBatchHandler) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 0, // Manual commit per batch
		StartOffset:    kafka.LastOffset,
	})

	return &KafkaConsumer{
		reader:  reader,
		handler: handler,
	}
}

// Start begins the event consumption loop in the background.
func (c *KafkaConsumer) Start(ctx context.Context) {
	consumerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.wg.Add(1)
	go c.loop(consumerCtx)
}

func (c *KafkaConsumer) loop(ctx context.Context) {
	defer c.wg.Done()

	const batchCapacity = 100
	const flushInterval = 1 * time.Second

	batch := make([]ClickEvent, 0, batchCapacity)
	msgs := make([]kafka.Message, 0, batchCapacity)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := c.handler(flushCtx, batch)
		cancel()

		if err != nil {
			log.Printf("[Kafka Consumer] Error persisting batch of %d clicks: %v", len(batch), err)
		} else {
			commitCtx, commitCancel := context.WithTimeout(context.Background(), 3*time.Second)
			if commitErr := c.reader.CommitMessages(commitCtx, msgs...); commitErr != nil {
				log.Printf("[Kafka Consumer] Commit error: %v", commitErr)
			}
			commitCancel()

			metrics.KafkaEventsConsumedTotal.Add(float64(len(batch)))
		}

		batch = batch[:0]
		msgs = msgs[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		default:
			// Fetch next message with short timeout to allow select ticker to tick
			fetchCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			msg, err := c.reader.FetchMessage(fetchCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
					flush()
					return
				}
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				log.Printf("[Kafka Consumer] Fetch error (retrying): %v", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			var event ClickEvent
			if err := json.Unmarshal(msg.Value, &event); err == nil {
				batch = append(batch, event)
				msgs = append(msgs, msg)
			}

			if len(batch) >= batchCapacity {
				flush()
			}
		}
	}
}

// Close stops the consumer loop and closes the underlying Kafka reader.
func (c *KafkaConsumer) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

