package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/internal/domain"
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

func (b BunMediaRepository) Get(ctx context.Context, id domain.MediaId) (*domain.Media, error) {
	var m Media
	err := b.db.NewSelect().Model(&m).Where("id = ?", uuid.UUID(id)).Relation("ExternalIds").Limit(1).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return m.toCore(), nil
}

func (b BunMediaRepository) GetByExternalID(ctx context.Context, id domain.ExternalId) (*domain.Media, error) {
	var m Media

	err := b.db.NewSelect().
		Model(&m).
		Where("EXISTS (SELECT 1 FROM external_ids WHERE media_id = media.id AND provider = ? AND value = ?)", id.Provider, id.Id).
		Relation("ExternalIds").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return m.toCore(), nil
}

func (b BunMediaRepository) Store(ctx context.Context, media *domain.Media) error {
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

func (b BunMediaRepository) StoreMediaWithItems(ctx context.Context, m domain.MediaWithItems) error {
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

func (b BunMediaRepository) GetMediaIdByExternalId(ctx context.Context, id domain.ExternalId) (*domain.MediaId, error) {
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
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return new(domain.MediaId(mediaID)), nil
}

func (b BunMediaRepository) List(ctx context.Context, q domain.MediaQuery) ([]domain.Media, int, error) {
	var dbMedia []Media
	var total int

	countQuery := b.db.NewSelect().
		Model((*Media)(nil)).
		Where("deleted_at IS NULL")

	if q.Type != "" {
		countQuery = countQuery.Where("type = ?", q.Type)
	}

	c, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	total = c

	selectQuery := b.db.NewSelect().
		Model(&dbMedia).
		Relation("ExternalIds"). //not sure if it's needed here, but will leave it as is
		Where("deleted_at IS NULL")

	if q.Type != "" {
		selectQuery = selectQuery.Where("type = ?", q.Type)
	}

	//sort
	sortBy := domain.SortByCreatedAt
	switch q.SortBy {
	case domain.SortByTitle:
		sortBy = "title"
	case domain.SortByUpdatedAt:
		sortBy = "updated_at"
	case domain.SortByLastSync:
		sortBy = "last_sync"
	case domain.SortByStatus:
		sortBy = "status"
	case domain.SortByType:
		sortBy = "type"
	case domain.SortByCreatedAt:
		sortBy = "created_at"
	}

	sortDir := "ASC"
	if q.SortDir == domain.SortDesc {
		sortDir = "DESC"
	}

	selectQuery = selectQuery.OrderExpr(string(sortBy) + " " + sortDir)

	//pagination
	if q.Paginate.Limit > 0 {
		selectQuery = selectQuery.Limit(q.Paginate.Limit)
	}
	if q.Paginate.Offset > 0 {
		selectQuery = selectQuery.Offset(q.Paginate.Offset)
	}

	if err := selectQuery.Scan(ctx); err != nil {
		return nil, 0, err
	}

	//map
	mediaList := make([]domain.Media, len(dbMedia))
	for i, m := range dbMedia {
		mediaList[i] = *m.toCore()
	}

	return mediaList, total, nil
}
