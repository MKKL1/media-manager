package metadata

import (
	"context"
	"fmt"
	"server/internal/core"
)

type SearchService struct {
}

// Search searches across all providers that support the given media type.
func (s *SearchService) Search(ctx context.Context, query string, mediaType core.MediaType) (interface{}, error) {
	if query == "" {
		return nil, fmt.Errorf("query required: %w", core.ErrInvalidInput)
	}

	//for _, p := range s.registry.All() {
	//	switch mediaType {
	//	case "movie":
	//		if searcher, ok := p.(movie.Searcher); ok {
	//			return searcher.SearchMovie(ctx, movie.SearchQuery{Title: query})
	//		}
	//	case "tv":
	//		if searcher, ok := p.(tv.Searcher); ok {
	//			return searcher.SearchTV(ctx, tv.SearchQuery{Title: query})
	//		}
	//	}
	//}

	return nil, fmt.Errorf("no provider for media type %s: %w", mediaType, core.ErrNoProvider)
}
