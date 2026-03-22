package workflows

import (
	"context"
	"errors"
	"server/internal/core"
	"server/internal/metadata"

	"github.com/cschleiden/go-workflows/worker"
	"github.com/cschleiden/go-workflows/workflow"
)

type MediaPullInput struct {
	ExtID     core.ExternalId `json:"ext_id"`
	MediaType core.MediaType  `json:"media_type"`
}

type MediaPullResult struct {
	MediaID string `json:"media_id"`
	Existed bool   `json:"existed"`
}

type pullActivities struct {
	repo     metadata.MediaRepository
	handlers metadata.Handlers
}

func (a *pullActivities) fetchMetadata(ctx context.Context, extID core.ExternalId, mediaType core.MediaType) (core.MediaWithItems, error) {
	handler, err := a.handlers.Get(mediaType)
	if err != nil {
		return core.MediaWithItems{}, err
	}
	result, err := handler.FetchMedia(ctx, extID)
	if err != nil {
		return core.MediaWithItems{}, err
	}
	return *result, nil
}

func RegisterPullWorkflow(
	w *worker.Worker,
	repo metadata.MediaRepository,
	handlers metadata.Handlers,
) func(workflow.Context, MediaPullInput) (MediaPullResult, error) {
	acts := &pullActivities{repo: repo, handlers: handlers}

	wf := func(ctx workflow.Context, input MediaPullInput) (MediaPullResult, error) {
		existing, err := acts.repo.GetByExternalID(context.Background(), input.ExtID)
		if err != nil && !errors.Is(err, core.ErrNotFound) {
			return MediaPullResult{}, err
		}
		if existing != nil {
			return MediaPullResult{MediaID: existing.ID.String(), Existed: true}, nil
		}

		fetchOpts := workflow.ActivityOptions{
			RetryOptions: workflow.RetryOptions{MaxAttempts: 3},
		}
		media, err := workflow.ExecuteActivity[core.MediaWithItems](
			ctx, fetchOpts, acts.fetchMetadata, input.ExtID, input.MediaType,
		).Get(ctx)
		if err != nil {
			return MediaPullResult{}, err
		}

		if err := acts.repo.StoreMediaWithItems(context.Background(), media); err != nil {
			return MediaPullResult{}, err
		}

		return MediaPullResult{MediaID: media.Media.ID.String()}, nil
	}

	w.RegisterWorkflow(wf)
	w.RegisterActivity(acts.fetchMetadata)

	return wf
}
