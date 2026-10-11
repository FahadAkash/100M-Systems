package shortener

import (
	"context"
	"testing"
	"time"

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

func TestService_CustomAliasAndExpiration(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	svc := NewService(store)

	// 1. Valid custom alias
	rec, err := svc.ShortenWithOptions(ctx, "https://golang.org", "go-lang", nil)
	if err != nil {
		t.Fatalf("expected nil err on custom alias, got %v", err)
	}
	if rec.Code != "go-lang" {
		t.Errorf("expected code go-lang, got %s", rec.Code)
	}

	// 2. Custom alias collision
	_, err = svc.ShortenWithOptions(ctx, "https://google.com", "go-lang", nil)
	if err != ErrAliasTaken {
		t.Fatalf("expected ErrAliasTaken, got %v", err)
	}

	// 3. Invalid custom alias format (too short)
	_, err = svc.ShortenWithOptions(ctx, "https://google.com", "ab", nil)
	if err != ErrInvalidAlias {
		t.Fatalf("expected ErrInvalidAlias for too short, got %v", err)
	}

	// 4. Invalid custom alias characters
	_, err = svc.ShortenWithOptions(ctx, "https://google.com", "bad*alias!", nil)
	if err != ErrInvalidAlias {
		t.Fatalf("expected ErrInvalidAlias for bad chars, got %v", err)
	}

	// 5. Expired URL resolution
	pastTime := time.Now().UTC().Add(-1 * time.Hour)
	expRec, err := svc.ShortenWithOptions(ctx, "https://example.com/expired", "expired-link", &pastTime)
	if err != nil {
		t.Fatalf("expected nil err on creating expired record, got %v", err)
	}

	_, err = svc.Resolve(ctx, expRec.Code)
	if err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound for expired link, got %v", err)
	}
}
