package database

import (
	"database/sql"
	"encoding/json"
	"server/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MediaIdentity struct {
	bun.BaseModel `bun:"table:external_ids"` //TODO rename

	MediaID uuid.UUID `bun:"media_id,type:uuid,notnull"`
	Kind    string    `bun:"kind,pk"`
	Value   string    `bun:"value,pk"`
}

type MediaImage struct {
	bun.BaseModel `bun:"table:media_images"`

	ID      uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	MediaID uuid.UUID `bun:"media_id,type:uuid,notnull"`
	Role    string    `bun:"role,notnull"`
	Source  string    `bun:"source,notnull"`
	Path    string    `bun:"path,notnull"`
}

type MediaItem struct {
	bun.BaseModel `bun:"table:media_items"`

	ID        uuid.UUID       `bun:"id,pk,type:uuid"`
	MediaID   uuid.UUID       `bun:"media_id,type:uuid,notnull"`
	Monitored bool            `bun:"monitored,notnull"`
	Status    string          `bun:"status,notnull"`
	Metadata  json.RawMessage `bun:"metadata,type:jsonb"`
	//TODO episode numbers should be a column to allow for queries on them I think
	// Maybe it could be done well enough with json queries and postgres
}

type Media struct {
	bun.BaseModel `bun:"table:media"`

	ID         uuid.UUID           `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Type       string              `bun:"type,notnull"`
	Title      string              `bun:"title,notnull"`
	Status     string              `bun:"status,notnull"`
	Monitored  bool                `bun:"monitored,notnull"`
	SourceKind string              `bun:"source_kind,notnull"`
	SourceID   string              `bun:"source_id,notnull"`
	Metadata   json.RawMessage     `bun:"metadata,type:jsonb,notnull"`
	CreatedAt  time.Time           `bun:"created_at,notnull"`
	LastSync   time.Time           `bun:"last_sync,notnull"`
	UpdatedAt  time.Time           `bun:"updated_at,notnull"`
	DeletedAt  sql.Null[time.Time] `bun:"deleted_at"`

	ExternalIDs []MediaIdentity `bun:"rel:has-many,join:id=media_id"`
	Images      []MediaImage    `bun:"rel:has-many,join:id=media_id"`
}

func (m *Media) toCore() *domain.Media {
	identities := make([]domain.MediaIdentity, len(m.ExternalIDs))
	for i, ext := range m.ExternalIDs {
		identities[i] = domain.MediaIdentity{
			Kind: domain.SourceKindFromString(ext.Kind),
			ID:   ext.Value,
		}
	}

	images := make([]domain.Image, len(m.Images))
	for i, img := range m.Images {
		images[i] = domain.Image{
			ID:           img.ID,
			Role:         domain.ImageRole(img.Role),
			Source:       domain.ProviderName(img.Source),
			ExternalPath: img.Path,
		}
	}

	return &domain.Media{
		ID:                domain.MediaID(m.ID),
		Type:              domain.MediaType(m.Type),
		Title:             m.Title,
		Status:            m.Status,
		Monitored:         m.Monitored,
		PrimaryIdentity:   domain.NewMediaIdentity(domain.SourceKindFromString(m.SourceKind), m.SourceID),
		RelatedIdentities: identities,
		Images:            images,
		Metadata:          m.Metadata,
		CreatedAt:         m.CreatedAt,
		LastSync:          m.LastSync,
		UpdatedAt:         m.UpdatedAt,
	}
}

func fromCore(c *domain.Media) *Media {
	externalIDs := make([]MediaIdentity, len(c.RelatedIdentities))
	for i, id := range c.RelatedIdentities {
		externalIDs[i] = MediaIdentity{
			MediaID: uuid.UUID(c.ID),
			Kind:    id.Kind.String(),
			Value:   id.ID,
		}
	}

	images := make([]MediaImage, len(c.Images))
	for i, img := range c.Images {
		images[i] = MediaImage{
			ID:      img.ID,
			MediaID: uuid.UUID(c.ID),
			Role:    string(img.Role),
			Source:  string(img.Source),
			Path:    img.ExternalPath,
		}
	}

	return &Media{
		ID:          uuid.UUID(c.ID),
		Type:        string(c.Type),
		Title:       c.Title,
		Status:      c.Status,
		Monitored:   c.Monitored,
		SourceKind:  c.PrimaryIdentity.Kind.String(),
		SourceID:    c.PrimaryIdentity.ID,
		ExternalIDs: externalIDs,
		Images:      images,
		Metadata:    c.Metadata,
		CreatedAt:   c.CreatedAt,
		LastSync:    c.LastSync,
		UpdatedAt:   c.UpdatedAt,
	}
}

func fromCoreItem(c domain.MediaItem) MediaItem {
	return MediaItem{
		ID:        c.ID,
		MediaID:   uuid.UUID(c.MediaId),
		Monitored: c.Monitored,
		Status:    string(c.Status),
		Metadata:  c.Metadata,
	}
}
