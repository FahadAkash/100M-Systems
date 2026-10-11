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

const codeLength = 8

var (
	ErrCodeGeneration = errors.New("could not generate unique code")
	ErrInvalidCode    = errors.New("invalid short code")
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

func (s *Service) Shorten(
	ctx context.Context,
	rawURL string,
) (storage.URL, error) {
	rawURL = strings.TrimSpace(rawURL)

	if err := ValidateURL(rawURL); err != nil {
		return storage.URL{}, err
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
	if len(code) != codeLength {
		return storage.URL{}, ErrInvalidCode
	}

	for _, r := range code {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_') {
			return storage.URL{}, ErrInvalidCode
		}
	}

	return s.store.GetByCode(ctx, code)
}

func generateCode() (string, error) {
	data := make([]byte, 6)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	// Six random bytes encode to eight URL-safe Base64 characters.
	return base64.RawURLEncoding.EncodeToString(data), nil
}
