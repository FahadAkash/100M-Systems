package storage

import (
	"context"
	"errors"
	"time"

	"github.com/FahadAkash/100M-Systems/pkg/events"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrExpired  = errors.New("url has expired")
)

type URL struct {
	Code      string     `json:"code"`
	Original  string     `json:"original_url"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type Store interface {
	// Save saves a URL to the storage.
	Save(ctx context.Context, url URL) error
	GetByCode(ctx context.Context, code string) (URL, error)
}

// AnalyticsStore defines methods for storing and retrieving click analytics.
type AnalyticsStore interface {
	SaveClickBatch(ctx context.Context, batch []events.ClickEvent) error
	GetAnalytics(ctx context.Context, code string) (events.ClickAnalyticsSummary, error)
}

type Storage = Store

