package postgres

import (
	"context"
	"errors"
	"fmt"
	"url-shortener/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

func New(storagePath string) (*Storage, error) {
	const fn = "storage.postgres.New"

	// Подключение к новой БД shortener
	db, err := pgxpool.New(context.Background(), storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}

	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("%s: connect to database: %w", fn, err)
	}
	return &Storage{db: db}, nil
}

const pgUniqueViolationCode = "23505"

func (s *Storage) SaveURL(ctx context.Context, urlToSave string, alias string) (int64, error) {
	const fn = "storage.postgres.SaveURL"

	var id int64
	err := s.db.QueryRow(
		ctx,
		`INSERT INTO url(alias, url) VALUES($1, $2) RETURNING id`,
		alias,
		urlToSave,
	).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode {
			return 0, fmt.Errorf("%s: %w", fn, storage.ErrURLExists)
		}
		return 0, fmt.Errorf("%s: %w", fn, err)
	}
	return id, nil
}

func (s *Storage) GetURL(ctx context.Context, alias string) (string, error) {
	const fn = "storage.postgres.GetURL"

	var url string
	err := s.db.QueryRow(ctx,
		"SELECT url FROM url WHERE alias = $1", alias).Scan(&url) // .Scan(&url) необходим для извлечения данных из результата SQL-запроса.

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}
		return "", fmt.Errorf("%s: execute statement %w", fn, err)
	}

	return url, nil
}

func (s *Storage) DeleteURL(ctx context.Context, alias string) (int64, error) {
	const fn = "storage.postgres.DeleteURL"

	result, err := s.db.Exec(ctx,
		"DELETE FROM url WHERE alias = $1", alias)
	if err != nil {
		return 0, fmt.Errorf("%s: execute statement %w", fn, err)
	}

	rowsAffected := result.RowsAffected() // Считаем сколько удалили

	return rowsAffected, nil
}

func (s *Storage) UpdateURL(ctx context.Context, alias string, newURL string) (int64, error) {
	const fn = "storage.postgres.UpdateURL"

	result, err := s.db.Exec(ctx,
		"UPDATE url SET url = $1 WHERE alias = $2", newURL, alias)

	if err != nil {
		return 0, fmt.Errorf("%s: execute statement %w", fn, err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected, nil
}

func (s *Storage) Close() error {
	s.db.Close()
	return nil
}
