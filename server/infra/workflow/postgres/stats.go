package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cschleiden/go-workflows/backend"
	"github.com/cschleiden/go-workflows/core"
	"github.com/cschleiden/go-workflows/workflow"
)

func (pb *postgresBackend) GetStats(ctx context.Context) (_ *backend.Stats, retErr error) {
	s := &backend.Stats{}

	tx, err := pb.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone && retErr == nil {
			retErr = rbErr
		}
	}()

	row := tx.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM instances i WHERE i.completed_at IS NULL",
	)
	if err := row.Err(); err != nil {
		return nil, fmt.Errorf("failed to query active instances: %w", err)
	}

	var activeInstances int64
	if err := row.Scan(&activeInstances); err != nil {
		return nil, fmt.Errorf("failed to scan active instances: %w", err)
	}

	s.ActiveWorkflowInstances = activeInstances

	now := time.Now()
	workflowRows, err := tx.QueryContext(
		ctx,
		`SELECT i.queue, COUNT(*)
			FROM instances i
			INNER JOIN pending_events pe ON i.instance_id = pe.instance_id
			WHERE
				i.state = $1 AND i.completed_at IS NULL
				AND (pe.visible_at IS NULL OR pe.visible_at <= $2)
				AND (i.locked_until IS NULL OR i.locked_until < $3)
			GROUP BY i.queue`,
		core.WorkflowInstanceStateActive,
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query active instances: %w", err)
	}
	defer func() {
		if cErr := workflowRows.Close(); cErr != nil && retErr == nil {
			retErr = cErr
		}
	}()

	s.PendingWorkflowTasks = make(map[core.Queue]int64)

	for workflowRows.Next() {
		var queue string
		var pendingInstances int64
		if err := workflowRows.Scan(&queue, &pendingInstances); err != nil {
			return nil, fmt.Errorf("failed to scan active instances: %w", err)
		}

		s.PendingWorkflowTasks[workflow.Queue(queue)] = pendingInstances
	}

	if err := workflowRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read active instances: %w", err)
	}

	activityRows, err := tx.QueryContext(
		ctx,
		"SELECT queue, COUNT(*) FROM activities GROUP BY queue")
	if err != nil {
		return nil, fmt.Errorf("failed to query active activities: %w", err)
	}
	defer func() {
		if cErr := activityRows.Close(); cErr != nil && retErr == nil {
			retErr = cErr
		}
	}()

	s.PendingActivityTasks = make(map[core.Queue]int64)

	for activityRows.Next() {
		var queue string
		var pendingActivities int64
		if err := activityRows.Scan(&queue, &pendingActivities); err != nil {
			return nil, fmt.Errorf("failed to scan active activities: %w", err)
		}

		s.PendingActivityTasks[workflow.Queue(queue)] = pendingActivities
	}

	if err := activityRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read active activities: %w", err)
	}

	return s, nil
}
