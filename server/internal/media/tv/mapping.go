package tv

import (
	"context"
	"server/internal/domain"
)

type SeasonMapping struct {
	sourceId     domain.MediaIdentity //anidb id
	seasonNumber int
	provider     string
}

type SeasonMappingRepository interface {
	FindSeasonMapping(
		ctx context.Context,
		id domain.MediaIdentity, //tmdb id
	) ([]SeasonMapping, error)
}
