package events

import (
	"net/url"
	"strings"
	"time"
)

// ClickEvent represents an asynchronous redirection/click payload for event streaming.
type ClickEvent struct {
	Code        string    `json:"code"`
	OriginalURL string    `json:"original_url"`
	IP          string    `json:"ip"`
	UserAgent   string    `json:"user_agent"`
	Referer     string    `json:"referer"`
	Timestamp   time.Time `json:"timestamp"`
}

// ClickAnalyticsSummary represents aggregated click metrics for a shortened URL.
type ClickAnalyticsSummary struct {
	Code             string              `json:"code"`
	OriginalURL      string              `json:"original_url"`
	TotalClicks      int64               `json:"total_clicks"`
	LastClicked      *time.Time          `json:"last_clicked,omitempty"`
	Browsers         map[string]int64    `json:"browsers"`
	OperatingSystems map[string]int64    `json:"operating_systems"`
	Devices          map[string]int64    `json:"devices"`
	TopReferrers     map[string]int64    `json:"top_referrers"`
	RecentClicks     []ClickEventSummary `json:"recent_clicks"`
}

// ClickEventSummary is a lightweight record of an individual click.
type ClickEventSummary struct {
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	Timestamp time.Time `json:"timestamp"`
}

// ParseUserAgent classifies raw User-Agent into browser, OS, and device categories.
func ParseUserAgent(ua string) (browser, os, device string) {
	if ua == "" {
		return "Unknown", "Unknown", "Unknown"
	}
	uaLower := strings.ToLower(ua)

	// Device & Bot Detection
	if strings.Contains(uaLower, "bot") || strings.Contains(uaLower, "crawler") || strings.Contains(uaLower, "spider") {
		device = "Bot"
	} else if strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "tablet") {
		device = "Tablet"
	} else if strings.Contains(uaLower, "mobile") || strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "android") {
		device = "Mobile"
	} else {
		device = "Desktop"
	}

	// Operating System Detection
	switch {
	case strings.Contains(uaLower, "windows"):
		os = "Windows"
	case strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "ios"):
		os = "iOS"
	case strings.Contains(uaLower, "macintosh") || strings.Contains(uaLower, "mac os"):
		os = "macOS"
	case strings.Contains(uaLower, "android"):
		os = "Android"
	case strings.Contains(uaLower, "linux"):
		os = "Linux"
	default:
		os = "Other"
	}

	// Browser Detection (Order matters due to overlapping tokens)
	switch {
	case strings.Contains(uaLower, "edg"):
		browser = "Edge"
	case strings.Contains(uaLower, "opr") || strings.Contains(uaLower, "opera"):
		browser = "Opera"
	case strings.Contains(uaLower, "chrome") || strings.Contains(uaLower, "crios"):
		browser = "Chrome"
	case strings.Contains(uaLower, "firefox") || strings.Contains(uaLower, "fxios"):
		browser = "Firefox"
	case strings.Contains(uaLower, "safari") && !strings.Contains(uaLower, "chrome"):
		browser = "Safari"
	case strings.Contains(uaLower, "curl") || strings.Contains(uaLower, "postman"):
		browser = "API Client"
	default:
		browser = "Other"
	}

	return browser, os, device
}

// NormalizeReferrer extracts a clean readable referrer source.
func NormalizeReferrer(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "Direct"
	}

	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return "Other"
	}

	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")

	switch {
	case strings.Contains(host, "google."):
		return "Google"
	case strings.Contains(host, "t.co") || strings.Contains(host, "twitter.") || strings.Contains(host, "x.com"):
		return "Twitter / X"
	case strings.Contains(host, "facebook.") || strings.Contains(host, "l.facebook."):
		return "Facebook"
	case strings.Contains(host, "linkedin."):
		return "LinkedIn"
	case strings.Contains(host, "reddit."):
		return "Reddit"
	case strings.Contains(host, "youtube."):
		return "YouTube"
	case strings.Contains(host, "github."):
		return "GitHub"
	default:
		return host
	}
}


