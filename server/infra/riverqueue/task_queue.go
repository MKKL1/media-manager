package riverqueue

import (
	"context"
	"server/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type TaskQueue struct {
	client *river.Client[pgx.Tx]
}

func NewTaskQueue(client *river.Client[pgx.Tx]) *TaskQueue {
	return &TaskQueue{client: client}
}

func (q *TaskQueue) Enqueue(ctx context.Context, args domain.JobArgs) (*domain.Job, error) {
	res, err := q.client.Insert(ctx, args, &river.InsertOpts{
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
			ByState: []rivertype.JobState{
				rivertype.JobStatePending,
				rivertype.JobStateAvailable,
				rivertype.JobStateRunning,
				rivertype.JobStateRetryable,
				rivertype.JobStateScheduled,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return &domain.Job{
		Id:        res.Job.ID,
		Duplicate: res.UniqueSkippedAsDuplicate,
	}, nil
}
