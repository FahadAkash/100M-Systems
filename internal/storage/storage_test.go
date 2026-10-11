package storage

import (
	"context"
	"testing"
	"time"
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

