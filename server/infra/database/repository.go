package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/internal/domain"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var _ domain.MediaRepository = (*MediaRepository)(nil)

type MediaRepository struct {
	db *bun.DB
}

func NewMediaRepository(db *bun.DB) *MediaRepository {
	return &MediaRepository{db}
}

func (r *MediaRepository) Get(ctx context.Context, id domain.MediaID) (*domain.Media, error) {
	var m Media
	err := r.db.NewSelect().
		Model(&m).
		Where("id = ?", uuid.UUID(id)).
		Relation("ExternalIDs").
		Relation("Images").
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

func (r *MediaRepository) GetByIdentity(ctx context.Context, id domain.MediaIdentity) (*domain.Media, error) {
	var m Media
	err := r.db.NewSelect().
		Model(&m).
		Where(
			"(source_kind = ? AND source_id = ?) OR "+
				"EXISTS (SELECT 1 FROM external_ids WHERE media_id = media.id AND kind = ? AND value = ?)",
			id.Kind.String(), id.ID, id.Kind.String(), id.ID,
		).
		Relation("ExternalIDs").
		Relation("Images").
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

func (r *MediaRepository) Store(ctx context.Context, media *domain.Media) error {
	m := fromCore(media)
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return upsertMedia(ctx, tx, m)
	})
}

func (r *MediaRepository) StoreWithItems(ctx context.Context, media *domain.Media, items []domain.MediaItem) error {
	m := fromCore(media)
	dbItems := make([]MediaItem, len(items))
	for i, item := range items {
		dbItems[i] = fromCoreItem(item)
	}

	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := upsertMedia(ctx, tx, m); err != nil {
			return err
		}
		if len(dbItems) > 0 {
			_, err := tx.NewInsert().
				Model(&dbItems).
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

func (r *MediaRepository) List(ctx context.Context, q domain.MediaQuery) ([]domain.Media, int, error) {
	var dbMedia []Media
	var total int

	countQuery := r.db.NewSelect().
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

	selectQuery := r.db.NewSelect().
		Model(&dbMedia).
		Relation("ExternalIDs"). //not sure if it's needed here, but will leave it as is
		Relation("Images").
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

func upsertMedia(ctx context.Context, tx bun.Tx, m *Media) error {
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

	if len(m.ExternalIDs) > 0 {
		_, err = tx.NewInsert().
			Model(&m.ExternalIDs).
			On("CONFLICT (kind, value) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("upsert external ids: %w", err)
		}
	}

	if len(m.Images) > 0 {
		_, err = tx.NewInsert().
			Model(&m.Images).
			On("CONFLICT (id) DO UPDATE").
			Set("role = EXCLUDED.role").
			Set("source = EXCLUDED.source").
			Set("path = EXCLUDED.path").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("upsert images: %w", err)
		}
	}

	return nil
}
