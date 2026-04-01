package postgres

import (
	"database/sql"
	"time"

	"github.com/cschleiden/go-workflows/backend"
)

type options struct {
	*backend.Options

	PostgresOptions func(db *sql.DB)

	ApplyMigrations     bool
	EnableNotifications bool

	// NotificationDSN is required when using NewPostgresBackendWithDB with
	// notifications enabled. With NewPostgresBackend it defaults to the
	// main connection DSN.
	NotificationDSN string

	// NotificationFallbackTimeout is the max time to wait for a LISTEN/NOTIFY
	// notification before falling back to a poll. Default: 30s.
	NotificationFallbackTimeout time.Duration
}

type option func(*options)

func WithApplyMigrations(applyMigrations bool) option {
	return func(o *options) {
		o.ApplyMigrations = applyMigrations
	}
}

func WithPostgresOptions(f func(db *sql.DB)) option {
	return func(o *options) {
		o.PostgresOptions = f
	}
}

func WithBackendOptions(opts ...backend.BackendOption) option {
	return func(o *options) {
		for _, opt := range opts {
			opt(o.Options)
		}
	}
}

func WithNotifications(enable bool) option {
	return func(o *options) {
		o.EnableNotifications = enable
	}
}

func WithNotificationDSN(dsn string) option {
	return func(o *options) {
		o.NotificationDSN = dsn
	}
}

func WithNotificationFallbackTimeout(d time.Duration) option {
	return func(o *options) {
		o.NotificationFallbackTimeout = d
	}
}
