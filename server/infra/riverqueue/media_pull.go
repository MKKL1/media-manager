package riverqueue

import (
	"context"
	"errors"
	"server/internal/core"
	"server/internal/metadata/jobs"
	"server/internal/metadata/services"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog"
)

type PullMediaWorker struct {
	river.WorkerDefaults[jobs.MediaPullArgs]
	service *services.PullService
	logger  zerolog.Logger
}

func NewPullMediaWorker(service *services.PullService, logger zerolog.Logger) *PullMediaWorker {
	return &PullMediaWorker{service: service, logger: logger}
}

func (w *PullMediaWorker) Work(ctx context.Context, job *river.Job[jobs.MediaPullArgs]) error {
	ctx = w.logger.WithContext(ctx)
	_, err := w.service.Pull(ctx, job.Args.ExtID, job.Args.MediaType)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) ||
			errors.Is(err, core.ErrAlreadyExists) ||
			errors.Is(err, core.ErrInvalidInput) ||
			errors.Is(err, core.ErrNoProvider) {
			return river.JobCancel(err)
		}
		return err
	}
	return nil
}
