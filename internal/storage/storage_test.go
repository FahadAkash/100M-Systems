package storage

import (
	"context"
	"testing"
	"time"

	"github.com/FahadAkash/100M-Systems/pkg/events"
)

func TestMemoryStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	url := URL{
		Code:      "abc12345",
		Original:  "https://example.com/long/path",
		CreatedAt: time.Now().UTC(),
	}

	// 1. Save
	if err := store.Save(ctx, url); err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}

	// 2. Conflict on duplicate save
	if err := store.Save(ctx, url); err != ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	// 3. Get existing
	got, err := store.GetByCode(ctx, "abc12345")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if got.Original != url.Original {
		t.Errorf("got %q, want %q", got.Original, url.Original)
	}

	// 4. Get non-existent
	if _, err := store.GetByCode(ctx, "nonexist"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_Analytics(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	url := URL{
		Code:      "track123",
		Original:  "https://example.com/target",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Save(ctx, url); err != nil {
		t.Fatalf("unexpected save err: %v", err)
	}

	clicks := []events.ClickEvent{
		{
			Code:        "track123",
			OriginalURL: url.Original,
			IP:          "192.168.1.10",
			UserAgent:   "Go-http-client/1.1",
			Referer:     "https://news.ycombinator.com",
			Timestamp:   time.Now().UTC().Add(-1 * time.Minute),
		},
		{
			Code:        "track123",
			OriginalURL: url.Original,
			IP:          "192.168.1.20",
			UserAgent:   "Mozilla/5.0",
			Referer:     "https://google.com",
			Timestamp:   time.Now().UTC(),
		},
	}

	if err := store.SaveClickBatch(ctx, clicks); err != nil {
		t.Fatalf("SaveClickBatch err: %v", err)
	}

	analytics, err := store.GetAnalytics(ctx, "track123")
	if err != nil {
		t.Fatalf("GetAnalytics err: %v", err)
	}

	if analytics.TotalClicks != 2 {
		t.Errorf("expected 2 clicks, got %d", analytics.TotalClicks)
	}
	if len(analytics.RecentClicks) != 2 {
		t.Errorf("expected 2 recent clicks, got %d", len(analytics.RecentClicks))
	}
	if analytics.RecentClicks[0].IP != "192.168.1.20" {
		t.Errorf("expected most recent click first, got IP %s", analytics.RecentClicks[0].IP)
	}
}


