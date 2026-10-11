package storage

import (
	"context"
	"sync"

	"github.com/FahadAkash/100M-Systems/pkg/events"
)

type MemoryStore struct {
	mu     sync.RWMutex
	urls   map[string]URL
	clicks map[string][]events.ClickEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		urls:   make(map[string]URL),
		clicks: make(map[string][]events.ClickEvent),
	}
}

func (m *MemoryStore) Save(
	ctx context.Context,
	url URL,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.urls == nil {
		m.urls = make(map[string]URL)
	}

	if _, exists := m.urls[url.Code]; exists {
		return ErrConflict
	}

	m.urls[url.Code] = url
	return nil
}

func (m *MemoryStore) GetByCode(
	ctx context.Context,
	code string,
) (URL, error) {
	if err := ctx.Err(); err != nil {
		return URL{}, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.urls != nil {
		if rec, exists := m.urls[code]; exists {
			return rec, nil
		}
	}

	return URL{}, ErrNotFound
}

// SaveClickBatch records a batch of click events into memory.
func (m *MemoryStore) SaveClickBatch(ctx context.Context, batch []events.ClickEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clicks == nil {
		m.clicks = make(map[string][]events.ClickEvent)
	}

	for _, click := range batch {
		m.clicks[click.Code] = append(m.clicks[click.Code], click)
	}

	return nil
}

// GetAnalytics computes the click analytics summary from in-memory click records.
func (m *MemoryStore) GetAnalytics(ctx context.Context, code string) (events.ClickAnalyticsSummary, error) {
	if err := ctx.Err(); err != nil {
		return events.ClickAnalyticsSummary{}, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	url, exists := m.urls[code]
	if !exists {
		return events.ClickAnalyticsSummary{}, ErrNotFound
	}

	clicks := m.clicks[code]
	summary := events.ClickAnalyticsSummary{
		Code:         code,
		OriginalURL:  url.Original,
		TotalClicks:  int64(len(clicks)),
		RecentClicks: make([]events.ClickEventSummary, 0),
	}

	if len(clicks) > 0 {
		lastTime := clicks[len(clicks)-1].Timestamp
		summary.LastClicked = &lastTime

		// Return up to 20 most recent clicks in reverse chronological order
		limit := 20
		count := 0
		for i := len(clicks) - 1; i >= 0 && count < limit; i-- {
			c := clicks[i]
			summary.RecentClicks = append(summary.RecentClicks, events.ClickEventSummary{
				IP:        c.IP,
				UserAgent: c.UserAgent,
				Referer:   c.Referer,
				Timestamp: c.Timestamp,
			})
			count++
		}
	}

	return summary, nil
}

