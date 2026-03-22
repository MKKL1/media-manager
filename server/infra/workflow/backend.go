package workflow

import (
	"database/sql"
	"time"

	"github.com/cschleiden/go-workflows/backend"
	"github.com/cschleiden/go-workflows/backend/postgres"
)

func NewBackend(host string, port int, user, password, database string) backend.Backend {
	return postgres.NewPostgresBackend(host, port, user, password, database,
		postgres.WithPostgresOptions(func(db *sql.DB) {
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(10)
			db.SetConnMaxLifetime(5 * time.Minute)
			db.SetConnMaxIdleTime(5 * time.Minute)
		}))
}
