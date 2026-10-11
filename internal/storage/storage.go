package storage

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type URL struct {
	Code      string    `json:"code"`
	Original  string    `json:"original_url"`
	CreatedAt time.Time `json:"created_at"`
}

type Store interface {
	// Save saves a URL to the storage.
	Save(ctx context.Context, url URL) error
	GetByCode(ctx context.Context, code string) (URL, error)
}

type Storage = Store
