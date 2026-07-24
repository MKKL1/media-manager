package metadata

import (
	"context"
	"server/internal/domain"
)

// MappingSource loads cross-reference data from an external dataset.
type MappingSource interface {
	Name() string
	Load(ctx context.Context, lastVersion string) (*MappingData, error)
}

type SearchProvider interface {
	Search(ctx context.Context, query domain.SearchQuery) ([]domain.SearchResult, error)
}

//
//type Refresher interface {
//	ShouldRefresh(media domain.Media) bool
//	RefreshMedia(ctx context.Context, media domain.Media) (*domain.MediaWithItems, error)
//}
