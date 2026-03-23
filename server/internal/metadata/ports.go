package metadata

import (
	"context"
	"fmt"
	"server/internal/domain"
)

// MediaHandler fetches metadata for a specific media type.
// Implemented by movie.Handler, tv.Handler.
type MediaHandler interface {
	FetchMedia(ctx context.Context, id domain.ExternalId) (*domain.MediaWithItems, error)
}

type Handlers map[domain.MediaType]MediaHandler

func (h Handlers) Get(t domain.MediaType) (MediaHandler, error) {
	handler, ok := h[t]
	if !ok {
		return nil, fmt.Errorf("unsupported media type %q: %w", t, domain.ErrInvalidInput)
	}
	return handler, nil
}

// MediaRepository persists media entities.
type MediaRepository interface {
	Get(ctx context.Context, id domain.MediaId) (*domain.Media, error)
	GetByExternalID(ctx context.Context, id domain.ExternalId) (*domain.Media, error)
	StoreMediaWithItems(ctx context.Context, m domain.MediaWithItems) error
}

// MappingSource loads cross-reference data from an external dataset.
type MappingSource interface {
	Name() string
	Load(ctx context.Context, lastVersion string) (*MappingData, error)
}

// MappingRepository stores and queries provider ID mappings.
type MappingRepository interface {
	ReplaceMappings(ctx context.Context, source string, ids []IDRow, seasons []SeasonRow) error
	GetSourceVersion(ctx context.Context, source string) (string, error)
	SetSourceVersion(ctx context.Context, source string, version string) error
	FindMappings(ctx context.Context, id domain.ExternalId) ([]domain.ExternalId, error)
}
