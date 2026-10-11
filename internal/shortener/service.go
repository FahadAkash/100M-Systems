package shortener

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/FahadAkash/100M-Systems/internal/storage"
)

const defaultCodeLength = 8

var (
	ErrCodeGeneration = errors.New("could not generate unique code")
	ErrInvalidCode    = errors.New("invalid short code")
	ErrAliasTaken     = errors.New("custom alias is already in use")
	ErrInvalidAlias   = errors.New("invalid custom alias: must be 3-32 alphanumeric characters, hyphens or underscores")
)

type Store interface {
	Save(ctx context.Context, url storage.URL) error
	GetByCode(ctx context.Context, code string) (storage.URL, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Shorten creates a shortened URL with default random 8-character code and no expiry.
func (s *Service) Shorten(
	ctx context.Context,
	rawURL string,
) (storage.URL, error) {
	return s.ShortenWithOptions(ctx, rawURL, "", nil)
}

// ShortenWithOptions creates a shortened URL supporting optional custom vanity alias and expiration time.
func (s *Service) ShortenWithOptions(
	ctx context.Context,
	rawURL string,
	customCode string,
	expiresAt *time.Time,
) (storage.URL, error) {
	rawURL = strings.TrimSpace(rawURL)

	if err := ValidateURL(rawURL); err != nil {
		return storage.URL{}, err
	}

	customCode = strings.TrimSpace(customCode)
	if customCode != "" {
		if !isValidCode(customCode) {
			return storage.URL{}, ErrInvalidAlias
		}

		record := storage.URL{
			Code:      customCode,
			Original:  rawURL,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: expiresAt,
		}

		err := s.store.Save(ctx, record)
		if err != nil {
			if errors.Is(err, storage.ErrConflict) {
				return storage.URL{}, ErrAliasTaken
			}
			return storage.URL{}, err
		}
		return record, nil
	}

	for attempt := 0; attempt < 5; attempt++ {
		code, err := generateCode()
		if err != nil {
			return storage.URL{}, err
		}

		record := storage.URL{
			Code:      code,
			Original:  rawURL,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: expiresAt,
		}

		err = s.store.Save(ctx, record)
		if err == nil {
			return record, nil
		}

		if !errors.Is(err, storage.ErrConflict) {
			return storage.URL{}, err
		}
	}

	return storage.URL{}, ErrCodeGeneration
}

func (s *Service) Resolve(
	ctx context.Context,
	code string,
) (storage.URL, error) {
	if !isValidCode(code) {
		return storage.URL{}, ErrInvalidCode
	}

	record, err := s.store.GetByCode(ctx, code)
	if err != nil {
		return storage.URL{}, err
	}

	// Check if URL has expired
	if record.ExpiresAt != nil && time.Now().UTC().After(*record.ExpiresAt) {
		return storage.URL{}, storage.ErrNotFound
	}

	return record, nil
}

func isValidCode(code string) bool {
	if len(code) < 3 || len(code) > 32 {
		return false
	}
	for _, r := range code {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func generateCode() (string, error) {
	data := make([]byte, 6)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	// Six random bytes encode to eight URL-safe Base64 characters.
	return base64.RawURLEncoding.EncodeToString(data), nil
}
