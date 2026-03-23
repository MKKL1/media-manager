package tv

import (
	"context"
	"server/internal/domain"
)

type SeasonMapping struct {
	sourceId     domain.ExternalId //anidb id
	seasonNumber int
	provider     string
}

type SeasonMappingRepository interface {
	FindSeasonMapping(
		ctx context.Context,
		id domain.ExternalId, //tmdb id
	) ([]SeasonMapping, error)
}
