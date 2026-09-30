package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"url-shortener/internal/storage"

	"github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const fn = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}

	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS url (
		id INTEGER PRIMARY KEY,
		alias TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(ctx context.Context, urlToSave string, alias string) (int64, error) {
	const fn = "storage.sqlite.SaveURL"

	res, err := s.db.ExecContext(ctx, "INSERT INTO url(url, alias) VALUES(?, ?)", urlToSave, alias)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", fn, storage.ErrURLExists)
		}

		return 0, fmt.Errorf("%s: %w", fn, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", fn, err)
	}

	return id, nil
}

func (s *Storage) GetURL(ctx context.Context, alias string) (string, error) {
	const fn = "storage.sqlite.GetURL"

	var url string
	err := s.db.QueryRowContext(ctx, "SELECT url FROM url WHERE alias = ?", alias).Scan(&url) // .Scan(&url) необходим для извлечения данных из результата SQL-запроса.

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}
		return "", fmt.Errorf("%s: execute statement %w", fn, err)
	}

	return url, nil
}

func (s *Storage) DeleteURL(ctx context.Context, alias string) (int64, error) {
	const fn = "storage.sqlite.DeleteURL"

	result, err := s.db.ExecContext(ctx, "DELETE FROM url WHERE alias = ?", alias)
	if err != nil {
		return 0, fmt.Errorf("%s: execute statement %w", fn, err)
	}

	rowsAffected, err := result.RowsAffected() // Считаем сколько удалили
	if err != nil {
		return 0, fmt.Errorf("%s: get rows affected: %w", fn, err) // Возвращаем 0 и ошибку
	}

	return rowsAffected, nil
}

func (s *Storage) UpdateURL(ctx context.Context, alias string, newURL string) (int64, error) {
	result, err := s.db.ExecContext(ctx, "UPDATE url SET url = ? WHERE alias = ?", newURL, alias)
	if err != nil {
		return 0, fmt.Errorf("storage.sqlite.UpdateURL: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("storage.sqlite.UpdateURL: %w", err)
	}
	return rows, nil
}

func (s *Storage) Close() error { return s.db.Close() }
