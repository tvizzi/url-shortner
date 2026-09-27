package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	// Создание таблиц в новой БД
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS url (
			id SERIAL PRIMARY KEY,
			alias TEXT NOT NULL UNIQUE,
			url TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}
	return &Storage{db: db}, nil
}

const pgUniqueViolationCode = "23505"

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const fn = "storage.postgres.SaveURL"

	var id int64
	err := s.db.QueryRow(
		context.Background(),
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

func (s *Storage) GetURL(alias string) (string, error) {
	const fn = "storage.postgres.GetURL"

	var url string
	err := s.db.QueryRow(context.Background(),
		"SELECT url FROM url WHERE alias = $1", alias).Scan(&url) // .Scan(&url) необходим для извлечения данных из результата SQL-запроса.

	if err != nil {
		log.Printf("Database error: %v", err) // Debug log
		if errors.Is(err, pgx.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}
		return "", fmt.Errorf("%s: execute statement %w", fn, err)
	}

	return url, nil
}

func (s *Storage) DeleteURL(alias string) (int64, error) {
	const fn = "storage.postgres.DeleteURL"

	result, err := s.db.Exec(context.Background(),
		"DELETE FROM url WHERE alias = $1", alias)
	if err != nil {
		return 0, fmt.Errorf("%s: execute statement %w", fn, err)
	}

	rowsAffected := result.RowsAffected() // Считаем сколько удалили

	return rowsAffected, nil
}

func (s *Storage) UpdateURL(alias string, newURL string) (int64, error) {
	const fn = "storage.postgres.UpdateURL"

	result, err := s.db.Exec(context.Background(),
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
