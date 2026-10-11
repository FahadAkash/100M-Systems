package storage

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu   sync.RWMutex
	urls map[string]URL
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		urls: make(map[string]URL),
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
