package tv

import (
	"context"
	"server/internal/core"
)

type AnimeMappingService struct {
	repo SeasonMappingRepository
}

func (r *AnimeMappingService) GetSeasonsById(ctx context.Context, id core.ExternalId) ([]SeasonMapping, error) {
	mapping, err := r.repo.FindSeasonMapping(ctx, id)
	if err != nil {
		return nil, err
	}

	return mapping, nil
}
