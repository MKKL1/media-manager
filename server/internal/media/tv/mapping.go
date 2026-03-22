package tv

import (
	"context"
	"server/internal/core"
)

type SeasonMapping struct {
	sourceId     core.ExternalId //anidb id
	seasonNumber int
	provider     string
}

type SeasonMappingRepository interface {
	FindSeasonMapping(
		ctx context.Context,
		id core.ExternalId, //tmdb id
	) ([]SeasonMapping, error)
}
