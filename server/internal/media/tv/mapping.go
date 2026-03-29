package tv

import (
	"context"
	"server/internal/domain"
)

type SeasonMapping struct {
	sourceId     domain.SourceID //anidb id
	seasonNumber int
	provider     string
}

type SeasonMappingRepository interface {
	FindSeasonMapping(
		ctx context.Context,
		id domain.SourceID, //tmdb id
	) ([]SeasonMapping, error)
}
