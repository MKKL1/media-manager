package services

import (
	"context"
	"fmt"
	"server/internal/core"

	"github.com/rs/zerolog"
)

type LibraryService struct {
	identity    MediaIdentityService
	pullService *PullService
}

func NewLibraryService(identity MediaIdentityService, pullService *PullService) *LibraryService {
	return &LibraryService{
		identity:    identity,
		pullService: pullService,
	}
}

func (s *LibraryService) Add(
	ctx context.Context,
	externalId core.ExternalId,
	mediaType core.MediaType,
) (*core.MediaId, string, error) {
	logger := zerolog.Ctx(ctx)

	existingID, err := s.identity.Resolve(ctx, externalId)
	if err != nil {
		return nil, "", err
	}
	if existingID != nil {
		logger.Debug().Str("media_id", existingID.String()).Msg("already in library")
		return existingID, "", nil
	}

	workflowID, err := s.pullService.RequestPull(ctx, externalId, mediaType)
	if err != nil {
		return nil, "", fmt.Errorf("request pull: %w", err)
	}

	logger.Debug().Str("workflow_id", workflowID).Msg("pull workflow started")
	return nil, workflowID, nil
}

func (s *LibraryService) Refresh(
	ctx context.Context,
	mediaID core.MediaId,
) (string, error) {
	return "", fmt.Errorf("not implemented")
}
