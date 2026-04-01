package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	workflowTasksChannel = "workflow_tasks"
	activityTasksChannel = "activity_tasks"

	defaultFallbackTimeout = 30 * time.Second
	reconnectDelay         = 10 * time.Second
)

type notificationListener struct {
	dsn             string
	logger          *slog.Logger
	fallbackTimeout time.Duration

	mu         sync.Mutex
	workflowCh chan struct{}
	activityCh chan struct{}
	closedCh   chan struct{}

	started bool
	closed  bool

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newNotificationListener(dsn string, logger *slog.Logger, fallbackTimeout time.Duration) *notificationListener {
	if fallbackTimeout <= 0 {
		fallbackTimeout = defaultFallbackTimeout
	}
	return &notificationListener{
		dsn:             dsn,
		logger:          logger,
		fallbackTimeout: fallbackTimeout,
		workflowCh:      make(chan struct{}),
		activityCh:      make(chan struct{}),
		closedCh:        make(chan struct{}),
	}
}

func (nl *notificationListener) Start(ctx context.Context) error {
	nl.mu.Lock()
	defer nl.mu.Unlock()

	if nl.closed {
		return fmt.Errorf("notification listener has been closed")
	}
	if nl.started {
		return nil
	}

	nl.ctx, nl.cancel = context.WithCancel(context.Background())

	conn, err := nl.connect(ctx)
	if err != nil {
		nl.cancel()
		return err
	}

	nl.started = true

	nl.wg.Add(1)
	go nl.listenLoop(conn)

	return nil
}

func (nl *notificationListener) Close() error {
	nl.mu.Lock()
	if nl.closed {
		nl.mu.Unlock()
		return nil
	}
	nl.closed = true
	close(nl.closedCh)
	if nl.cancel != nil {
		nl.cancel()
	}
	nl.mu.Unlock()

	nl.wg.Wait()
	return nil
}

func (nl *notificationListener) connect(ctx context.Context) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, nl.dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting for notifications: %w", err)
	}

	if _, err := conn.Exec(ctx, "LISTEN "+workflowTasksChannel); err != nil {
		if cErr := conn.Close(context.Background()); cErr != nil {
			nl.logger.Error("closing connection after LISTEN workflow_tasks failure", "error", cErr)
		}
		return nil, fmt.Errorf("LISTEN %s: %w", workflowTasksChannel, err)
	}
	if _, err := conn.Exec(ctx, "LISTEN "+activityTasksChannel); err != nil {
		if cErr := conn.Close(context.Background()); cErr != nil {
			nl.logger.Error("closing connection after LISTEN activity_tasks failure", "error", cErr)
		}
		return nil, fmt.Errorf("LISTEN %s: %w", activityTasksChannel, err)
	}

	nl.logger.Debug("notification listener connected",
		"channels", []string{workflowTasksChannel, activityTasksChannel})

	return conn, nil
}

func (nl *notificationListener) listenLoop(conn *pgx.Conn) {
	defer nl.wg.Done()

	for {
		err := nl.processNotifications(conn)
		if cErr := conn.Close(context.Background()); cErr != nil {
			nl.logger.Error("closing notification connection", "error", cErr)
		}

		if nl.ctx.Err() != nil {
			return
		}

		nl.logger.Error("notification listener disconnected, reconnecting", "error", err)

		for {
			select {
			case <-nl.ctx.Done():
				return
			case <-time.After(reconnectDelay):
			}

			var connectErr error
			conn, connectErr = nl.connect(nl.ctx)
			if connectErr != nil {
				if nl.ctx.Err() != nil {
					return
				}
				nl.logger.Error("notification listener reconnect failed", "error", connectErr)
				continue
			}
			nl.logger.Info("notification listener reconnected")
			break
		}
	}
}

func (nl *notificationListener) processNotifications(conn *pgx.Conn) error {
	for {
		notification, err := conn.WaitForNotification(nl.ctx)
		if err != nil {
			return err
		}

		switch notification.Channel {
		case workflowTasksChannel:
			nl.broadcastWorkflow()
		case activityTasksChannel:
			nl.broadcastActivity()
		}
	}
}

func (nl *notificationListener) broadcastWorkflow() {
	nl.mu.Lock()
	ch := nl.workflowCh
	nl.workflowCh = make(chan struct{})
	nl.mu.Unlock()
	close(ch)
}

func (nl *notificationListener) broadcastActivity() {
	nl.mu.Lock()
	ch := nl.activityCh
	nl.activityCh = make(chan struct{})
	nl.mu.Unlock()
	close(ch)
}

func (nl *notificationListener) PrepareWaitForWorkflowTask() func(ctx context.Context) bool {
	nl.mu.Lock()
	ch := nl.workflowCh
	closedCh := nl.closedCh
	fallback := nl.fallbackTimeout
	nl.mu.Unlock()

	return func(ctx context.Context) bool {
		select {
		case <-ctx.Done():
			return false
		case <-closedCh:
			return false
		case <-ch:
			return true
		case <-time.After(fallback):
			return true
		}
	}
}

func (nl *notificationListener) PrepareWaitForActivityTask() func(ctx context.Context) bool {
	nl.mu.Lock()
	ch := nl.activityCh
	closedCh := nl.closedCh
	fallback := nl.fallbackTimeout
	nl.mu.Unlock()

	return func(ctx context.Context) bool {
		select {
		case <-ctx.Done():
			return false
		case <-closedCh:
			return false
		case <-ch:
			return true
		case <-time.After(fallback):
			return true
		}
	}
}
