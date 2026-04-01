package metadata

import (
	"context"
	"server/internal/domain"
)

// MediaHandler Every media type module must implement this
// Implemented by movie.Handler, tv.Handler.
type MediaHandler interface {
	Type() domain.MediaType
	// FetchMedia calls external metadata provider
	FetchMedia(ctx context.Context, id domain.MediaIdentity) (*domain.MediaWithItems, error)
	ToSummary(media domain.Media) (domain.MediaSummary, error)
}

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
