package database

import (
	"fmt"
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
	Monitored bool            `bun:"monitored"`
	Status    string          `bun:"status"`
	Metadata  json.RawMessage `bun:"metadata,type:jsonb"`
}

type Media struct {
	bun.BaseModel `bun:"table:media"`

	ID         uuid.UUID       `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Type       string          `bun:"type"`
	Title      string          `bun:"title"`
	Status     string          `bun:"status"`
	Monitored  bool            `bun:"monitored"`
	Provider   string          `bun:"provider,notnull"`
	ProviderId string          `bun:"provider_id,notnull"`
	Metadata   json.RawMessage `bun:"metadata,type:jsonb"`
	CreatedAt  time.Time       `bun:"created_at"`
	LastSync   time.Time       `bun:"last_sync"`
	UpdatedAt  time.Time       `bun:"updated_at"`
	DeletedAt  time.Time       `bun:"deleted_at"`

	ExternalIds []ExternalId `bun:"rel:has-many,join:id=media_id"`
}

func (m *Media) toCore() *domain.Media {
	externalIds := make([]domain.ExternalId, len(m.ExternalIds))
	for i, ext := range m.ExternalIds {
		externalIds[i] = domain.ExternalId{
			Provider: ext.Provider,
			Id:       ext.Value,
		}
	}

	return &domain.Media{
		ID:                domain.MediaId(m.ID),
		Type:              m.Type,
		Title:             m.Title,
		Status:            m.Status,
		Monitored:         m.Monitored,
		PrimaryExternalId: domain.NewExternalId(m.Provider, m.ProviderId),
		ExternalIds:       externalIds,
		Metadata:          m.Metadata,
		CreatedAt:         m.CreatedAt,
		LastSync:          m.LastSync,
		UpdatedAt:         m.UpdatedAt,
	}
}

func fromCore(c *domain.Media) (*Media, error) {
	externalIds := make([]ExternalId, len(c.ExternalIds))
	for i, ext := range c.ExternalIds {
		externalIds[i] = ExternalId{
			MediaID:  uuid.UUID(c.ID),
			Provider: ext.Provider,
			Value:    ext.Id,
		}
	}

	var metadataBytes json.RawMessage
	if c.Metadata != nil {
		if raw, ok := c.Metadata.(json.RawMessage); ok {
			metadataBytes = raw
		} else if b, ok := c.Metadata.([]byte); ok {
			metadataBytes = json.RawMessage(b)
		} else {
			b, err := json.Marshal(c.Metadata)
			if err != nil {
				return nil, fmt.Errorf("serialize media metadata: %w", err)
			}
			metadataBytes = b
		}
	}

	return &Media{
		ID:          uuid.UUID(c.ID),
		Type:        c.Type,
		Title:       c.Title,
		Status:      c.Status,
		Monitored:   c.Monitored,
		Provider:    c.PrimaryExternalId.Provider,
		ProviderId:  c.PrimaryExternalId.Id,
		ExternalIds: externalIds,
		Metadata:    metadataBytes,
		CreatedAt:   c.CreatedAt,
		LastSync:    c.LastSync,
		UpdatedAt:   c.UpdatedAt,
	}, nil
}

func fromCoreItem(c domain.MediaItem) (MediaItem, error) {
	var metadataBytes json.RawMessage
	if c.Metadata != nil {
		if raw, ok := c.Metadata.(json.RawMessage); ok {
			metadataBytes = raw
		} else if b, ok := c.Metadata.([]byte); ok {
			metadataBytes = b
		} else {
			b, err := json.Marshal(c.Metadata)
			if err != nil {
				return MediaItem{}, fmt.Errorf("serialize item metadata: %w", err)
			}
			metadataBytes = b
		}
	}

	return MediaItem{
		ID:        c.ID,
		MediaID:   uuid.UUID(c.MediaId),
		Monitored: c.Monitored,
		Status:    string(c.Status),
		Metadata:  metadataBytes,
	}, nil
}
