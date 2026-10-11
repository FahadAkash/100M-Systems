package shortener

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

var ErrInvalidURL = errors.New("invalid URL")

func ValidateURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ErrInvalidURL
	}

	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return ErrInvalidURL
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidURL
	}

	if u.Hostname() == "" {
		return ErrInvalidURL
	}

	if u.User != nil {
		return ErrInvalidURL
	}

	host := strings.ToLower(u.Hostname())

	// Reject localhost and literal IP addresses that are
	// clearly local/private. This is a basic URL policy,
	// not a complete SSRF defense.
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return ErrInvalidURL
	}

	if ip := net.ParseIP(host); ip != nil && ip.IsPrivate() {
		return ErrInvalidURL
	}

	return nil
}
