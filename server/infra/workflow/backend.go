package workflow

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/cschleiden/go-workflows/backend"
	"github.com/cschleiden/go-workflows/backend/postgres"
	"github.com/cschleiden/go-workflows/diag"
	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"
)

func NewBackend(host string, port int, user, password, database string, logger zerolog.Logger) diag.Backend {
	return postgres.NewPostgresBackend(host, port, user, password, database,
		postgres.WithBackendOptions(backend.WithLogger(slog.New(slogzerolog.Option{Logger: &logger}.NewZerologHandler()))),
		postgres.WithPostgresOptions(func(db *sql.DB) {
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(10)
			db.SetConnMaxLifetime(5 * time.Minute)
			db.SetConnMaxIdleTime(5 * time.Minute)
		}),
	)
}
