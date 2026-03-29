package metadata

import (
	"context"
	"server/internal/domain"
	"sort"
)

type SearchService struct {
	providers []SearchProvider
}

func NewSearchService(providers []SearchProvider) *SearchService {
	return &SearchService{providers: providers}
}

func (s *SearchService) Search(ctx context.Context, q domain.SearchQuery) ([]domain.SearchResult, error) {
	var all []domain.SearchResult
	for _, p := range s.providers {
		results, err := p.Search(ctx, q)
		if err != nil {
			return nil, err
		}
		all = append(all, results...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Popularity > all[j].Popularity
	})

	return all, nil
}
