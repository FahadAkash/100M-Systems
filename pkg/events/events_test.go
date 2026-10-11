package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNoopProducer(t *testing.T) {
	producer := &NoopProducer{}
	producer.Publish(ClickEvent{
		Code:        "test1234",
		OriginalURL: "https://example.com",
		Timestamp:   time.Now(),
	})

	if err := producer.Close(); err != nil {
		t.Fatalf("expected nil on NoopProducer.Close, got %v", err)
	}
}

func TestClickEvent_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	event := ClickEvent{
		Code:        "test1234",
		OriginalURL: "https://example.com/stream",
		IP:          "192.168.0.50",
		UserAgent:   "Curl/8.0",
		Referer:     "https://github.com",
		Timestamp:   now,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed ClickEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed.Code != event.Code || parsed.OriginalURL != event.OriginalURL || parsed.IP != event.IP {
		t.Errorf("mismatch: got %+v, want %+v", parsed, event)
	}
}

func TestKafkaProducer_NonBlockingBuffer(t *testing.T) {
	// Create producer with a tiny buffer of 2 to verify non-blocking drop behavior
	producer := NewKafkaProducer([]string{"127.0.0.1:9092"}, "test-topic", 2)
	defer producer.Close()

	// Rapidly emit 10 events to ensure it never blocks or deadlocks
	for i := 0; i < 10; i++ {
		producer.Publish(ClickEvent{
			Code:        "burst123",
			OriginalURL: "https://example.com",
			IP:          "127.0.0.1",
			Timestamp:   time.Now(),
		})
	}
}

