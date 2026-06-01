package tags

import (
	"context"
	"fmt"
	"strings"
)

const maxTagLength = 100

// Service implements business logic for tag management.
type Service struct {
	repo *Repository
}

// NewService creates a new Service instance.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// EnsureExists creates any tags that don't already exist in the database.
func (s *Service) EnsureExists(ctx context.Context, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	if err := s.validateTags(names); err != nil {
		return nil, err
	}

	names = s.normalizeTags(names)

	return names, s.repo.EnsureExists(ctx, names)
}

// validateTags validates a list of tag names.
// Tags must be non-empty after trimming and within max length.
func (s *Service) validateTags(tags []string) error {
	for i, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return fmt.Errorf("%w: tag at index %d is empty", ErrValidationFailed, i)
		}
		if len(tag) > maxTagLength {
			return fmt.Errorf("%w: tag at index %d exceeds maximum length of %d", ErrValidationFailed, i, maxTagLength)
		}
	}
	return nil
}

// normalizeTags cleans tag names: trims spaces, lowercases, deduplicates.
func (s *Service) normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		normalized := strings.TrimSpace(strings.ToLower(tag))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}
