package services

import (
	"context"
	"fmt"
	"server/internal/core"
	"server/internal/metadata/jobs"

	"github.com/rs/zerolog"
)

// LibraryService handles user-facing library operations.
type LibraryService struct {
	identity  MediaIdentityService
	taskQueue core.JobQueue
}

func (s *LibraryService) Add(
	ctx context.Context,
	externalId core.ExternalId,
	mediaType core.MediaType,
) (*core.MediaId, *core.Job, error) {
	logger := zerolog.Ctx(ctx)

	// Check if we already have this media
	existingID, err := s.identity.Resolve(ctx, externalId)
	if err != nil {
		return nil, nil, err
	}
	if existingID != nil {
		logger.Debug().Str("media_id", existingID.String()).Msg("already in library")
		return existingID, nil, nil
	}

	// Queue background pull
	task, err := s.taskQueue.Enqueue(ctx, jobs.MediaPullArgs{
		ExtID:     externalId,
		MediaType: mediaType,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("enqueue pull: %w", err)
	}

	logger.Debug().Int64("task_id", task.Id).Msg("pull queued")
	return nil, task, nil
}

func (s *LibraryService) Refresh(
	ctx context.Context,
	mediaID core.MediaId,
) (*core.Job, error) {
	// TODO: queue refresh task
	// would fetch latest metadata but preserve user overrides
	return nil, fmt.Errorf("not implemented")
}
