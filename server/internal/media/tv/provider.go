package tv

import (
	"context"
)

type Searcher interface {
	SearchTV(ctx context.Context, query SearchQuery) ([]SearchResult, error)
}

type Fetcher interface {
	GetShow(ctx context.Context, id string) (*ProviderShow, error)
	GetEpisodes(ctx context.Context, id string) ([]ProviderEpisode, error)
}
