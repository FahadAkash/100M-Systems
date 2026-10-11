package events

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FahadAkash/100M-Systems/pkg/metrics"
	"github.com/segmentio/kafka-go"
)

// ClickProducer defines the interface for emitting click events.
type ClickProducer interface {
	Publish(event ClickEvent)
	Close() error
}

// NoopProducer is used when Kafka is disabled.
type NoopProducer struct{}

func (n *NoopProducer) Publish(event ClickEvent) {}
func (n *NoopProducer) Close() error             { return nil }

// KafkaProducer manages asynchronous publishing of click events to Kafka.
type KafkaProducer struct {
	writer *kafka.Writer
	queue  chan ClickEvent
	done   chan struct{}
	closed atomic.Bool
	wg     sync.WaitGroup
}

// NewKafkaProducer initializes a Kafka writer with a dedicated non-blocking buffered queue.
func NewKafkaProducer(brokers []string, topic string, bufferSize int) *KafkaProducer {
	if bufferSize <= 0 {
		bufferSize = 5000
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		BatchSize:    100,
		BatchTimeout: 50 * time.Millisecond,
		Async:        false,
		RequiredAcks: kafka.RequireOne,
	}

	p := &KafkaProducer{
		writer: w,
		queue:  make(chan ClickEvent, bufferSize),
		done:   make(chan struct{}),
	}

	p.wg.Add(1)
	go p.worker()

	return p
}

// Publish enqueues a click event in a strictly non-blocking fashion.
// Under no circumstances will this stall or fail an active HTTP request.
func (p *KafkaProducer) Publish(event ClickEvent) {
	if p == nil || p.closed.Load() {
		return
	}

	select {
	case p.queue <- event:
		metrics.KafkaEventsProducedTotal.Inc()
	default:
		// Queue full - drop payload gracefully to safeguard response latency
		log.Printf("[Kafka Producer] Warning: event queue full, dropped click event for code=%s", event.Code)
	}
}

// worker drains the in-memory queue and flushes micro-batches to the Kafka topic.
func (p *KafkaProducer) worker() {
	defer p.wg.Done()

	batch := make([]kafka.Message, 0, 50)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := p.writer.WriteMessages(ctx, batch...)
		cancel()
		if err != nil {
			log.Printf("[Kafka Producer] Failed to write %d messages to Kafka: %v", len(batch), err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-p.done:
			// Drain remaining events in queue
			for len(p.queue) > 0 {
				event := <-p.queue
				if data, err := json.Marshal(event); err == nil {
					batch = append(batch, kafka.Message{
						Key:   []byte(event.Code),
						Value: data,
						Time:  event.Timestamp,
					})
				}
			}
			flush()
			return

		case event := <-p.queue:
			if data, err := json.Marshal(event); err == nil {
				batch = append(batch, kafka.Message{
					Key:   []byte(event.Code),
					Value: data,
					Time:  event.Timestamp,
				})
			}
			if len(batch) >= 50 {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

// Close shuts down the producer worker and cleanly flushes pending messages.
func (p *KafkaProducer) Close() error {
	if p == nil || !p.closed.CompareAndSwap(false, true) {
		return nil
	}

	close(p.done)
	p.wg.Wait()
	return p.writer.Close()
}

