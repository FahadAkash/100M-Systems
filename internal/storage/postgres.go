package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/FahadAkash/100M-Systems/pkg/events"
	"github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{
		db: db,
	}
}

func (p *PostgresStore) Save(ctx context.Context, url URL) error {
	_, err := p.db.ExecContext(
		ctx,
		`INSERT INTO urls (code , original_url , created_at)
		VALUES ($1 , $2 , $3)`,
		url.Code,
		url.Original,
		url.CreatedAt,
	)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
	}
	return err
}

func (p *PostgresStore) GetByCode(ctx context.Context, code string) (URL, error) {
	var url URL
	err := p.db.QueryRowContext(
		ctx,
		`SELECT code, original_url, created_at FROM urls WHERE code = $1`,
		code,
	).Scan(&url.Code, &url.Original, &url.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return url, ErrNotFound
		}
		return url, err
	}
	return url, nil
}

// SaveClickBatch inserts a batch of click events into the url_clicks table inside a single transaction.
func (p *PostgresStore) SaveClickBatch(ctx context.Context, batch []events.ClickEvent) error {
	if len(batch) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO url_clicks (code, original_url, ip, user_agent, referer, clicked_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, click := range batch {
		if _, err := stmt.ExecContext(ctx, click.Code, click.OriginalURL, click.IP, click.UserAgent, click.Referer, click.Timestamp); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetAnalytics queries the click metrics and recent clicks for a given short code.
func (p *PostgresStore) GetAnalytics(ctx context.Context, code string) (events.ClickAnalyticsSummary, error) {
	var summary events.ClickAnalyticsSummary
	summary.Code = code

	// 1. Verify URL existence
	err := p.db.QueryRowContext(ctx, `SELECT original_url FROM urls WHERE code = $1`, code).Scan(&summary.OriginalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return summary, ErrNotFound
		}
		return summary, err
	}

	// 2. Count total clicks and find latest click timestamp
	var lastClicked sql.NullTime
	err = p.db.QueryRowContext(ctx, `
		SELECT COUNT(*), MAX(clicked_at)
		FROM url_clicks
		WHERE code = $1
	`, code).Scan(&summary.TotalClicks, &lastClicked)
	if err != nil {
		return summary, err
	}
	if lastClicked.Valid {
		t := lastClicked.Time
		summary.LastClicked = &t
	}

	// 3. Retrieve up to 20 most recent clicks
	rows, err := p.db.QueryContext(ctx, `
		SELECT ip, user_agent, referer, clicked_at
		FROM url_clicks
		WHERE code = $1
		ORDER BY clicked_at DESC
		LIMIT 20
	`, code)
	if err != nil {
		return summary, err
	}
	defer rows.Close()

	summary.RecentClicks = make([]events.ClickEventSummary, 0)
	for rows.Next() {
		var item events.ClickEventSummary
		if err := rows.Scan(&item.IP, &item.UserAgent, &item.Referer, &item.Timestamp); err == nil {
			summary.RecentClicks = append(summary.RecentClicks, item)
		}
	}

	return summary, nil
}

