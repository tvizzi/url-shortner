package urlservice

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage"
)

const defaultAliasLength = 6
const maxGenerateAttempts = 5

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string { return e.Message }

func (e ValidationError) Unwrap() error { return storage.ErrInvalidInput }

type URLStorage interface {
	SaveURL(ctx context.Context, urlToSave string, alias string) (int64, error)
	GetURL(ctx context.Context, alias string) (string, error)
	DeleteURL(ctx context.Context, alias string) (int64, error)
	UpdateURL(ctx context.Context, alias string, newURL string) (int64, error)
}

type Service struct {
	storage     URLStorage
	aliasLength int
}

func New(storage URLStorage, aliasLength int) *Service {
	if aliasLength <= 0 {
		aliasLength = defaultAliasLength
	}

	return &Service{
		storage:     storage,
		aliasLength: aliasLength,
	}
}

func (s *Service) SaveURL(ctx context.Context, urlToSave string, alias string) (string, error) {
	if err := validateURL(urlToSave); err != nil {
		return "", err
	}
	if alias != "" && strings.TrimSpace(alias) == "" {
		return "", ValidationError{Message: "field Alias is not valid"}
	}
	if alias != "" {
		if _, err := s.storage.SaveURL(ctx, urlToSave, alias); err != nil {
			return "", err
		}

		return alias, nil
	}

	var lastErr error

	for i := 0; i < maxGenerateAttempts; i++ {
		generated := random.NewRandomString(s.aliasLength)

		if _, err := s.storage.SaveURL(ctx, urlToSave, generated); err != nil {
			if errors.Is(err, storage.ErrURLExists) {
				lastErr = err
				continue
			}

			return "", err
		}

		return generated, nil
	}

	return "", fmt.Errorf("failed to generate a unique alias after %d attempts: %w", maxGenerateAttempts, lastErr)
}

func (s *Service) GetURL(ctx context.Context, alias string) (string, error) {
	if strings.TrimSpace(alias) == "" {
		return "", ValidationError{Message: "field Alias is a required field"}
	}
	return s.storage.GetURL(ctx, alias)
}

func (s *Service) DeleteURL(ctx context.Context, alias string) (int64, error) {
	if strings.TrimSpace(alias) == "" {
		return 0, ValidationError{Message: "field Alias is a required field"}
	}
	rowsAffected, err := s.storage.DeleteURL(ctx, alias)
	if err != nil {
		return 0, err
	}

	if rowsAffected == 0 {
		return 0, storage.ErrURLNotFound
	}

	return rowsAffected, nil
}

func (s *Service) UpdateURL(ctx context.Context, alias string, newURL string) (int64, error) {
	if strings.TrimSpace(alias) == "" {
		return 0, ValidationError{Message: "field Alias is a required field"}
	}
	if err := validateURL(newURL); err != nil {
		return 0, err
	}
	rowsAffected, err := s.storage.UpdateURL(ctx, alias, newURL)
	if err != nil {
		return 0, err
	}

	if rowsAffected == 0 {
		return 0, storage.ErrURLNotFound
	}

	return rowsAffected, nil
}

func validateURL(value string) error {
	if strings.TrimSpace(value) == "" {
		return ValidationError{Message: "field URL is a required field"}
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ValidationError{Message: "field URL is not a valid URL"}
	}
	return nil
}
