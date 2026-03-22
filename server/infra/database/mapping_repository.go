package database

import (
	"context"
	"database/sql"
	"fmt"
	"server/internal/core"
	"server/internal/metadata"
	"time"

	"github.com/uptrace/bun"
)

type MappingSource struct {
	bun.BaseModel `bun:"table:mapping_source"`
	Id            int       `bun:"id,pk,autoincrement"`
	Name          string    `bun:"name,pk"`
	Version       string    `bun:"version"`
	SyncedAt      time.Time `bun:"synced_at"`
}

type ProviderMapping struct {
	bun.BaseModel `bun:"table:provider_mapping"`
	SourceId      int    `bun:"source_id,notnull"`
	GroupID       int    `bun:"group_id,notnull"`
	Provider      string `bun:"provider,pk"`
	ProviderID    string `bun:"provider_id,pk"`
}

type SeasonMapping struct {
	bun.BaseModel  `bun:"table:season_mapping"`
	SourceId       int    `bun:"source_id,notnull"`
	GroupID        int    `bun:"group_id,notnull"`
	Provider       string `bun:"provider,pk"`
	ProviderID     string `bun:"provider_id,pk"`
	TargetProvider string `bun:"target_provider,pk"`
	SeasonNumber   int    `bun:"season_number,notnull"`
}

type MappingRepository struct {
	db *bun.DB
}

func NewMappingRepository(db *bun.DB) *MappingRepository {
	return &MappingRepository{db: db}
}

func (r *MappingRepository) getOrCreateSource(ctx context.Context, tx bun.Tx, name string) (int, error) {
	src := &MappingSource{Name: name}
	_, err := tx.NewInsert().
		Model(src).
		On("CONFLICT (name) DO UPDATE").
		Set("name = EXCLUDED.name").
		Returning("id").
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return src.Id, nil
}

func (r *MappingRepository) ReplaceMappings(ctx context.Context, source string, ids []metadata.IDRow, seasons []metadata.SeasonRow) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	sourceID, err := r.getOrCreateSource(ctx, tx, source)
	if err != nil {
		return fmt.Errorf("get or create source: %w", err)
	}

	// Delete existing data for this source
	_, err = tx.NewDelete().Model((*ProviderMapping)(nil)).Where("source_id = ?", sourceID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete provider mappings: %w", err)
	}
	_, err = tx.NewDelete().Model((*SeasonMapping)(nil)).Where("source_id = ?", sourceID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete season mappings: %w", err)
	}

	// Batch insert provider mappings
	const batchSize = 3000

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := make([]ProviderMapping, end-i)
		for j, row := range ids[i:end] {
			batch[j] = ProviderMapping{
				SourceId:   sourceID,
				GroupID:    row.GroupID,
				Provider:   row.Provider,
				ProviderID: row.ProviderID,
			}
		}
		_, err := tx.NewInsert().Model(&batch).Exec(ctx)
		if err != nil {
			return fmt.Errorf("insert provider mappings batch %d: %w", i/batchSize, err)
		}
	}

	// Batch insert season mappings
	for i := 0; i < len(seasons); i += batchSize {
		end := i + batchSize
		if end > len(seasons) {
			end = len(seasons)
		}
		batch := make([]SeasonMapping, end-i)
		for j, row := range seasons[i:end] {
			batch[j] = SeasonMapping{
				SourceId:       sourceID,
				GroupID:        row.GroupID,
				Provider:       row.Provider,
				ProviderID:     row.ProviderID,
				TargetProvider: row.TargetProvider,
				SeasonNumber:   row.SeasonNumber,
			}
		}
		_, err := tx.NewInsert().Model(&batch).Exec(ctx)
		if err != nil {
			return fmt.Errorf("insert season mappings batch %d: %w", i/batchSize, err)
		}
	}

	return tx.Commit()
}

func (r *MappingRepository) GetSourceVersion(ctx context.Context, source string) (string, error) {
	var version string
	err := r.db.NewSelect().
		Model((*MappingSource)(nil)).
		Column("version").
		Where("name = ?", source).
		Scan(ctx, &version)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return version, nil
}

func (r *MappingRepository) SetSourceVersion(ctx context.Context, source string, version string) error {
	_, err := r.db.NewInsert().
		Model(&MappingSource{
			Name:     source,
			Version:  version,
			SyncedAt: time.Now(),
		}).
		On("CONFLICT (name) DO UPDATE").
		Set("version = EXCLUDED.version").
		Set("synced_at = EXCLUDED.synced_at").
		Exec(ctx)
	return err
}

// FindMappings returns all cross-referenced IDs for a given provider:id
func (r *MappingRepository) FindMappings(ctx context.Context, id core.ExternalId) ([]core.ExternalId, error) {
	var mappings []ProviderMapping
	err := r.db.NewSelect().
		Model(&mappings).
		Where("(source_id, group_id) IN (?)",
			r.db.NewSelect().
				Model((*ProviderMapping)(nil)).
				Column("source_id", "group_id").
				Where("provider = ? AND provider_id = ?", id.Provider, id.Id),
		).
		Where("NOT (provider = ? AND provider_id = ?)", id.Provider, id.Id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]core.ExternalId, len(mappings))
	for i, m := range mappings {
		result[i] = core.NewExternalId(m.Provider, m.ProviderID)
	}
	return result, nil
}

// FindSeasonMapping returns source entity IDs that map to a specific season
// e.g. "tmdb:62913 season 2" → returns [anidb:3]
func (r *MappingRepository) FindSeasonMapping(ctx context.Context, id core.ExternalId, targetProvider string, seasonNumber int) ([]core.ExternalId, error) {
	var mappings []SeasonMapping
	err := r.db.NewSelect().
		Model(&mappings).
		Where("(source_id, group_id) IN (?)",
			r.db.NewSelect().
				Model((*ProviderMapping)(nil)).
				Column("source_id", "group_id").
				Where("provider = ? AND provider_id = ?", id.Provider, id.Id),
		).
		Where("target_provider = ?", targetProvider).
		Where("season_number = ?", seasonNumber).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]core.ExternalId, len(mappings))
	for i, m := range mappings {
		result[i] = core.NewExternalId(m.Provider, m.ProviderID)
	}
	return result, nil
}
