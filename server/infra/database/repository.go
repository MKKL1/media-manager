package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/internal/core"
	"server/internal/metadata"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var _ metadata.MediaRepository = (*BunMediaRepository)(nil)

type BunMediaRepository struct {
	db *bun.DB
}

func NewBunMediaRepository(db *bun.DB) *BunMediaRepository {
	return &BunMediaRepository{db}
}

func (b BunMediaRepository) Get(ctx context.Context, id core.MediaId) (*core.Media, error) {
	var m Media
	err := b.db.NewSelect().Model(&m).Where("id = ?", uuid.UUID(id)).Relation("ExternalIds").Limit(1).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	return m.toCore(), nil
}

func (b BunMediaRepository) GetByExternalID(ctx context.Context, id core.ExternalId) (*core.Media, error) {
	var m Media

	err := b.db.NewSelect().
		Model(&m).
		Where("EXISTS (SELECT 1 FROM external_ids WHERE media_id = media.id AND provider = ? AND value = ?)", id.Provider, id.Id).
		Relation("ExternalIds").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	return m.toCore(), nil
}

func (b BunMediaRepository) Store(ctx context.Context, media *core.Media) error {
	m, err := fromCore(media)
	if err != nil {
		return err
	}

	return b.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(m).
			On("CONFLICT (id) DO UPDATE").
			Set("title = EXCLUDED.title").
			Set("status = EXCLUDED.status").
			Set("monitored = EXCLUDED.monitored").
			Set("metadata = EXCLUDED.metadata").
			Set("last_sync = EXCLUDED.last_sync").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("upsert media: %w", err)
		}

		if len(m.ExternalIds) > 0 {
			_, err = tx.NewInsert().
				Model(&m.ExternalIds).
				On("CONFLICT (provider, value) DO NOTHING").
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("upsert external ids: %w", err)
			}
		}

		return nil
	})
}

func (b BunMediaRepository) StoreMediaWithItems(ctx context.Context, m core.MediaWithItems) error {
	media, err := fromCore(&m.Media)
	if err != nil {
		return err
	}

	items := make([]MediaItem, len(m.Items))
	for i, item := range m.Items {
		dbItem, err := fromCoreItem(item)
		if err != nil {
			return err
		}
		items[i] = dbItem
	}

	return b.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(media).
			On("CONFLICT (id) DO UPDATE").
			Set("title = EXCLUDED.title").
			Set("status = EXCLUDED.status").
			Set("monitored = EXCLUDED.monitored").
			Set("metadata = EXCLUDED.metadata").
			Set("last_sync = EXCLUDED.last_sync").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("upsert media: %w", err)
		}

		if len(media.ExternalIds) > 0 {
			_, err = tx.NewInsert().
				Model(&media.ExternalIds).
				On("CONFLICT (provider, value) DO NOTHING").
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("upsert external ids: %w", err)
			}
		}

		if len(items) > 0 {
			_, err = tx.NewInsert().
				Model(&items).
				On("CONFLICT (id) DO UPDATE").
				Set("monitored = EXCLUDED.monitored").
				Set("status = EXCLUDED.status").
				Set("metadata = EXCLUDED.metadata").
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("upsert media items: %w", err)
			}
		}

		return nil
	})
}

func (b BunMediaRepository) GetMediaIdByExternalId(ctx context.Context, id core.ExternalId) (*core.MediaId, error) {
	var mediaID uuid.UUID
	err := b.db.NewSelect().
		TableExpr("external_ids").
		Column("media_id").
		Where("provider = ?", id.Provider).
		Where("value = ?", id.Id).
		Limit(1).
		Scan(ctx, &mediaID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	res := core.MediaId(mediaID)
	return &res, nil
}
