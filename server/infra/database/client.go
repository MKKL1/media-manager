package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bunotel"
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

	db := bun.NewDB(sqldb, pgdialect.New()).
		WithQueryHook(bunotel.NewQueryHook(
			bunotel.WithDBName("db"),
		))

	return db, nil
}

func Migrate(ctx context.Context, db *bun.DB) error {
	models := []interface{}{
		(*Media)(nil),
		(*ExternalId)(nil),
		(*MediaItem)(nil),
		(*MappingSource)(nil),
		(*ProviderMapping)(nil),
		(*SeasonMapping)(nil),
	}

	for _, model := range models {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_mapping_source_name ON mapping_source (name)`,
		`CREATE INDEX IF NOT EXISTS idx_provider_mapping_group ON provider_mapping (source_id, group_id) INCLUDE (provider, provider_id)`,
		`CREATE INDEX IF NOT EXISTS idx_season_mapping_group ON season_mapping (source_id, group_id, target_provider, season_number) INCLUDE (provider, provider_id)`,
	}

	for _, idx := range indexes {
		if _, err := db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}
