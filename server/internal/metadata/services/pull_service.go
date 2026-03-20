package services

import (
	"context"
	"errors"
	"fmt"
	"server/internal/core"
	"server/internal/metadata"
	"server/internal/metadata/jobs"

	"github.com/rs/zerolog"
)

// PullService executes metadata pulls. Called by workers.
type PullService struct {
	repo     metadata.MediaRepository
	handlers metadata.Handlers
	queue    core.JobQueue
}

func NewPullService(
	repo metadata.MediaRepository,
	queue core.JobQueue,
	handlers metadata.Handlers,
) *PullService {
	return &PullService{
		repo:     repo,
		queue:    queue,
		handlers: handlers,
	}
}

func (s *PullService) RequestPull(ctx context.Context, extID core.ExternalId, mediaType core.MediaType) (*core.Job, error) {
	job, err := s.queue.Enqueue(ctx, jobs.MediaPullArgs{
		ExtID:     extID,
		MediaType: mediaType,
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue pull job: %w", err)
	}

	return job, nil
}

func (s *PullService) Pull(
	ctx context.Context,
	extID core.ExternalId,
	mediaType core.MediaType,
) (*core.MediaId, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("ext_provider", extID.Provider).
		Str("ext_id", extID.Id).
		Logger()

	existing, err := s.repo.GetByExternalID(ctx, extID)
	if err != nil && !errors.Is(err, core.ErrNotFound) {
		return nil, fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		logger.Debug().Str("media_id", existing.ID.String()).Msg("already exists")
		return &existing.ID, nil
	}

	handler, err := s.handlers.Get(mediaType)
	if err != nil {
		return nil, fmt.Errorf("unknown media type %s", mediaType)
	}

	logger.Debug().Msg("fetching metadata from provider")
	media, err := handler.FetchMedia(ctx, extID)
	if err != nil {
		return nil, fmt.Errorf("fetch media: %w", err)
	}

	logger.Debug().Int("items_count", len(media.Items)).Msg("storing")
	err = s.repo.StoreMediaWithItems(ctx, *media)
	if err != nil {
		return nil, fmt.Errorf("store media: %w", err)
	}

	logger.Debug().Str("media_id", media.Media.ID.String()).Msg("complete")
	return &media.Media.ID, nil
}
