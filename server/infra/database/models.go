package database

import (
	"database/sql"
	"server/internal/domain"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ExternalId struct {
	bun.BaseModel `bun:"table:external_ids"`

	MediaID  uuid.UUID `bun:"media_id,type:uuid,notnull"`
	Provider string    `bun:"provider,pk"`
	Value    string    `bun:"value,pk"`
}

type MediaItem struct {
	bun.BaseModel `bun:"table:media_items"`

	ID        uuid.UUID       `bun:"id,pk,type:uuid"`
	MediaID   uuid.UUID       `bun:"media_id,type:uuid,notnull"`
	Monitored bool            `bun:"monitored,notnull"`
	Status    string          `bun:"status,notnull"`
	Metadata  json.RawMessage `bun:"metadata,type:jsonb"`
}

type MediaImage struct {
	bun.BaseModel `bun:"table:media_images"`

	ID       uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	MediaID  uuid.UUID `bun:"media_id,type:uuid,notnull"`
	Role     string    `bun:"role,notnull"`
	Provider string    `bun:"provider,notnull"`
	Path     string    `bun:"path,notnull"`
}

type Media struct {
	bun.BaseModel `bun:"table:media"`

	ID         uuid.UUID           `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Type       string              `bun:"type,notnull"`
	Title      string              `bun:"title,notnull"`
	Status     string              `bun:"status,notnull"`
	Monitored  bool                `bun:"monitored,notnull"`
	Provider   string              `bun:"provider,notnull"`
	ProviderId string              `bun:"provider_id,notnull"`
	Metadata   json.RawMessage     `bun:"metadata,type:jsonb,notnull"`
	CreatedAt  time.Time           `bun:"created_at,notnull"`
	LastSync   time.Time           `bun:"last_sync,notnull"`
	UpdatedAt  time.Time           `bun:"updated_at,notnull"`
	DeletedAt  sql.Null[time.Time] `bun:"deleted_at"` //TODO not sure what to do with it, but it could be useful to have "trash"

	ExternalIds []ExternalId `bun:"rel:has-many,join:id=media_id"`
	Images      []MediaImage `bun:"rel:has-many,join:id=media_id"`
}

func (m *Media) toCore() *domain.Media {
	externalIds := make([]domain.ExternalId, len(m.ExternalIds))
	for i, ext := range m.ExternalIds {
		externalIds[i] = domain.ExternalId{
			Provider: ext.Provider,
			Id:       ext.Value,
		}
	}

	images := make([]domain.Image, len(m.Images))
	for i, img := range m.Images {
		images[i] = domain.Image{
			ID:           img.ID,
			Role:         domain.ImageRole(img.Role),
			Provider:     img.Provider,
			ExternalPath: img.Path,
		}
	}

	return &domain.Media{
		ID:                domain.MediaId(m.ID),
		Type:              domain.MediaType(m.Type),
		Title:             m.Title,
		Status:            m.Status,
		Monitored:         m.Monitored,
		PrimaryExternalId: domain.NewExternalId(m.Provider, m.ProviderId),
		ExternalIds:       externalIds,
		Metadata:          m.Metadata,
		CreatedAt:         m.CreatedAt,
		LastSync:          m.LastSync,
		UpdatedAt:         m.UpdatedAt,
		Images:            images,
	}
}

// TODO remove error
func fromCore(c *domain.Media) (*Media, error) {
	externalIds := make([]ExternalId, len(c.ExternalIds))
	for i, ext := range c.ExternalIds {
		externalIds[i] = ExternalId{
			MediaID:  uuid.UUID(c.ID),
			Provider: ext.Provider,
			Value:    ext.Id,
		}
	}

	images := make([]MediaImage, len(c.Images))
	for i, img := range c.Images {
		images[i] = MediaImage{
			ID:       img.ID,
			MediaID:  uuid.UUID(c.ID),
			Role:     string(img.Role),
			Provider: img.Provider,
			Path:     img.ExternalPath,
		}
	}

	return &Media{
		ID:          uuid.UUID(c.ID),
		Type:        string(c.Type),
		Title:       c.Title,
		Status:      c.Status,
		Monitored:   c.Monitored,
		Provider:    c.PrimaryExternalId.Provider,
		ProviderId:  c.PrimaryExternalId.Id,
		ExternalIds: externalIds,
		Images:      images,
		Metadata:    c.Metadata,
		CreatedAt:   c.CreatedAt,
		LastSync:    c.LastSync,
		UpdatedAt:   c.UpdatedAt,
	}, nil
}

func fromCoreItem(c domain.MediaItem) (MediaItem, error) {

	return MediaItem{
		ID:        c.ID,
		MediaID:   uuid.UUID(c.MediaId),
		Monitored: c.Monitored,
		Status:    string(c.Status),
		Metadata:  c.Metadata,
	}, nil
}
