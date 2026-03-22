package services

import (
	"context"
	"errors"
	"fmt"
	"server/internal/core"
	"server/internal/metadata"

	"github.com/rs/zerolog"
)

// MediaIdentityService It aggregates many sources to know id of media in every available source
type MediaIdentityService struct {
	repo metadata.MediaRepository
}

//TODO maybe move calling external APIs for id discovery, so it works for example every 24h, but this seems like better fit

// Resolve Does everything it can to find every id for given media, this includes calling external APIs
// if null then media doesn't exist in database
// We do it to make sure that we don't query old data
// TODO better return
func (r *MediaIdentityService) Resolve(ctx context.Context, id core.ExternalId) (*core.MediaId, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("component", "MediaIdentityService").
		Str("ext_provider", id.Provider).
		Str("ext_id", id.Id).
		Logger()

	logger.Debug().Msg("resolving media identity")

	entityID, err := r.repo.GetMediaIdByExternalId(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			logger.Debug().Msg("identity not found in local database")
			// Not a real error, just means we don't have it yet
		} else {
			return nil, fmt.Errorf("get media id by external id: %w", err)
		}
	} else if entityID != nil {
		logger.Debug().Str("media_id", entityID.String()).Msg("identity resolved locally")
		return entityID, nil
	}

	//crossRefs, err := r.mappings.FindMappings(ctx, id)
	//if err != nil {
	//	return nil, err
	//}
	//for _, ref := range crossRefs {
	//	entityID, err = r.externalIDs.GetMediaIdByExternalId(ctx, ref)
	//	if err != nil {
	//		return nil, err
	//	}
	//	if entityID != nil {
	//		// Store the original ID so step 1 catches it next time
	//		_ = r.externalIDs.Store(ctx, *entityID, []ExternalId{id})
	//		return entityID, nil
	//	}
	//}

	return nil, nil
}
