package events

import "time"

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
	Code         string              `json:"code"`
	OriginalURL  string              `json:"original_url"`
	TotalClicks  int64               `json:"total_clicks"`
	LastClicked  *time.Time          `json:"last_clicked,omitempty"`
	RecentClicks []ClickEventSummary `json:"recent_clicks"`
}

// ClickEventSummary is a lightweight record of an individual click.
type ClickEventSummary struct {
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	Timestamp time.Time `json:"timestamp"`
}

