package metadata

import (
	"context"
	"server/internal/domain"
	"time"
)

// MappingRepository stores and queries provider ID mappings.
type MappingRepository interface {
	ReplaceMappings(ctx context.Context, source string, ids []IDRow, seasons []SeasonRow) error
	GetSourceVersion(ctx context.Context, source string) (string, time.Time, error)
	SetSourceVersion(ctx context.Context, source string, version string) error
	FindMappings(ctx context.Context, id domain.ExternalId) ([]domain.ExternalId, error)
	MarkAttempt(ctx context.Context, source string) error
}

// MediaRepository persists media entities.
type MediaRepository interface {
	Get(ctx context.Context, id domain.MediaId) (*domain.Media, error)
	GetByExternalID(ctx context.Context, id domain.ExternalId) (*domain.Media, error)
	StoreMediaWithItems(ctx context.Context, m domain.MediaWithItems) error
	List(ctx context.Context, q domain.MediaQuery) ([]domain.Media, int, error)
}
