package workflows

import (
	"context"
	"errors"
	"server/internal/core"
	"server/internal/metadata"

	"github.com/cschleiden/go-workflows/worker"
	"github.com/cschleiden/go-workflows/workflow"
)

// --- IO types ---

type MediaPullInput struct {
	ExtID     core.ExternalId `json:"ext_id"`
	MediaType core.MediaType  `json:"media_type"`
}

type MediaPullResult struct {
	MediaID string `json:"media_id"`
	Existed bool   `json:"existed"`
}

type existenceResult struct {
	Found   bool   `json:"found"`
	MediaID string `json:"media_id,omitempty"`
}

// --- Activities ---

type pullActivities struct {
	repo     metadata.MediaRepository
	handlers metadata.Handlers
}

// Package-level reference: set once at startup, read by the workflow function
// during execution and replay. Safe because registration happens before any
// workflow instance can run.
var pullActs *pullActivities

// RegisterPullWorkflow wires the workflow and its activities into the worker.
func RegisterPullWorkflow(w *worker.Worker, repo metadata.MediaRepository, handlers metadata.Handlers) {
	pullActs = &pullActivities{repo: repo, handlers: handlers}

	w.RegisterWorkflow(MediaPullWorkflow)
	w.RegisterActivity(pullActs.checkExistence)
	w.RegisterActivity(pullActs.fetchMetadata)
	w.RegisterActivity(pullActs.storeMedia)
}

// --- Workflow ---

func MediaPullWorkflow(ctx workflow.Context, input MediaPullInput) (MediaPullResult, error) {
	// 1. Short-circuit if already stored
	check, err := workflow.ExecuteActivity[existenceResult](
		ctx, workflow.DefaultActivityOptions,
		pullActs.checkExistence, input.ExtID,
	).Get(ctx)
	if err != nil {
		return MediaPullResult{}, err
	}
	if check.Found {
		return MediaPullResult{MediaID: check.MediaID, Existed: true}, nil
	}

	// 2. Fetch metadata from external provider (retries on transient errors)
	fetchOpts := workflow.ActivityOptions{
		RetryOptions: workflow.RetryOptions{
			MaxAttempts: 3,
		},
	}
	media, err := workflow.ExecuteActivity[core.MediaWithItems](
		ctx, fetchOpts,
		pullActs.fetchMetadata, input.ExtID, input.MediaType,
	).Get(ctx)
	if err != nil {
		return MediaPullResult{}, err
	}

	// 3. Persist
	_, err = workflow.ExecuteActivity[bool](
		ctx, workflow.DefaultActivityOptions,
		pullActs.storeMedia, media,
	).Get(ctx)
	if err != nil {
		return MediaPullResult{}, err
	}

	return MediaPullResult{MediaID: media.Media.ID.String()}, nil
}

// --- Activity implementations ---

func (a *pullActivities) checkExistence(ctx context.Context, extID core.ExternalId) (existenceResult, error) {
	m, err := a.repo.GetByExternalID(ctx, extID)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return existenceResult{}, nil
		}
		return existenceResult{}, err
	}
	return existenceResult{Found: true, MediaID: m.ID.String()}, nil
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

func (a *pullActivities) storeMedia(ctx context.Context, media core.MediaWithItems) (bool, error) {
	return true, a.repo.StoreMediaWithItems(ctx, media)
}
