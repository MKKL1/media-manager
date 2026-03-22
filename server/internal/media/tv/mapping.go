package tv

import (
	"context"
	"server/internal/core"
)

type SeasonMappingRepository interface {
	FindSeasonMapping(
		ctx context.Context,
		id core.ExternalId,
		targetProvider string,
		seasonNumber int,
	) ([]core.ExternalId, error)
}
