package urlservice

import (
	"errors"
	"fmt"

	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage"
)

const defaultAliasLength = 6
const maxGenerateAttempts = 5

type URLStorage interface {
	SaveURL(urlToSave string, alias string) (int64, error)
	GetURL(alias string) (string, error)
	DeleteURL(alias string) (int64, error)
	UpdateURL(alias string, newURL string) (int64, error)
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

func (s *Service) SaveURL(urlToSave string, alias string) (string, error) {
	if alias != "" {
		if _, err := s.storage.SaveURL(urlToSave, alias); err != nil {
			return "", err
		}

		return alias, nil
	}

	var lastErr error

	for i := 0; i < maxGenerateAttempts; i++ {
		generated := random.NewRandomString(s.aliasLength)

		if _, err := s.storage.SaveURL(urlToSave, generated); err != nil {
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

func (s *Service) GetURL(alias string) (string, error) {
	return s.storage.GetURL(alias)
}

func (s *Service) DeleteURL(alias string) (int64, error) {
	rowsAffected, err := s.storage.DeleteURL(alias)
	if err != nil {
		return 0, err
	}

	if rowsAffected == 0 {
		return 0, storage.ErrURLNotFound
	}

	return rowsAffected, nil
}

func (s *Service) UpdateURL(alias string, newURL string) (int64, error) {
	rowsAffected, err := s.storage.UpdateURL(alias, newURL)
	if err != nil {
		return 0, err
	}

	if rowsAffected == 0 {
		return 0, storage.ErrURLNotFound
	}

	return rowsAffected, nil
}
