package storage

import (
	"context"
	"database/sql"
	"errors"

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
