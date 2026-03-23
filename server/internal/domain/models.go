package domain

import (
	"time"

	"github.com/google/uuid"
)

type MediaType string

type MediaId uuid.UUID

func (id MediaId) UUID() uuid.UUID    { return uuid.UUID(id) }
func (id MediaId) String() string     { return uuid.UUID(id).String() }
func NewMediaID(id uuid.UUID) MediaId { return MediaId(id) }
func GenerateMediaID() MediaId        { return MediaId(uuid.New()) }

type ExternalId struct {
	Provider string `json:"provider"`
	Id       string `json:"id"`
}

func (id ExternalId) String() string               { return id.Provider + ":" + id.Id }
func NewExternalId(provider, id string) ExternalId { return ExternalId{provider, id} }

type ItemStatus string

type Media struct {
	ID                MediaId
	Type              string
	Title             string
	Status            string
	Monitored         bool
	PrimaryExternalId ExternalId
	ExternalIds       []ExternalId
	Metadata          any
	CreatedAt         time.Time
	LastSync          time.Time
	UpdatedAt         time.Time
}

type MediaItem struct {
	ID        uuid.UUID
	MediaId   MediaId
	Monitored bool
	Status    ItemStatus
	Metadata  any
}

type MediaWithItems struct {
	Media Media
	Items []MediaItem
}
