package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func NewDB(connectionString string) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(connectionString),
		pgdriver.WithInsecure(true),
	))

	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(10)
	sqldb.SetConnMaxLifetime(5 * time.Minute)
	sqldb.SetConnMaxIdleTime(5 * time.Minute)

	if err := sqldb.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return bun.NewDB(sqldb, pgdialect.New()), nil
}

func Migrate(ctx context.Context, db *bun.DB) error {
	models := []interface{}{
		(*Media)(nil),
		(*ExternalId)(nil),
		(*MediaItem)(nil),
	}

	for _, model := range models {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	return nil
}
