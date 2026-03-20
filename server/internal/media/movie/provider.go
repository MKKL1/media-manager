package movie

import (
	"context"
)

// Searcher searches external providers for movies.
type Searcher interface {
	SearchMovie(ctx context.Context, query SearchQuery) ([]SearchResult, error)
}

// Fetcher fetches full movie metadata from a provider by its provider-specific ID.
type Fetcher interface {
	GetMovie(ctx context.Context, id string) (*ProviderMovie, error)
}
