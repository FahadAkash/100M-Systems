package shortener

import (
	"context"
	"testing"

	"github.com/FahadAkash/100M-Systems/internal/storage"
)

func TestService_ShortenAndResolve(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	svc := NewService(store)

	// 1. Shorten valid URL
	rec, err := svc.Shorten(ctx, "https://github.com/FahadAkash/100M-Systems")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}

	if len(rec.Code) != 8 {
		t.Fatalf("expected 8 char code, got %d (%s)", len(rec.Code), rec.Code)
	}

	// 2. Resolve short code
	resolved, err := svc.Resolve(ctx, rec.Code)
	if err != nil {
		t.Fatalf("expected nil err on resolve, got %v", err)
	}
	if resolved.Original != "https://github.com/FahadAkash/100M-Systems" {
		t.Errorf("got %q, want original URL", resolved.Original)
	}

	// 3. Reject invalid URL
	if _, err := svc.Shorten(ctx, "not-a-valid-url"); err != ErrInvalidURL {
		t.Fatalf("expected ErrInvalidURL, got %v", err)
	}
}

