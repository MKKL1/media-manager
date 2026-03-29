package domain

import (
	"encoding/json"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
)

type MediaType string

type MediaID uuid.UUID

func (id MediaID) String() string     { return uuid.UUID(id).String() }
func NewMediaID(id uuid.UUID) MediaID { return MediaID(id) }
func GenerateMediaID() MediaID        { return MediaID(uuid.New()) }

type ItemStatus string

type Media struct {
	ID                MediaID
	Type              MediaType
	Title             string
	Status            string
	Monitored         bool
	PrimaryIdentity   MediaIdentity
	RelatedIdentities []MediaIdentity
	Images            []Image
	Metadata          json.RawMessage //This shouldn't be json, but instead some metadata object
	CreatedAt         time.Time
	LastSync          time.Time
	UpdatedAt         time.Time
}

type MediaItem struct {
	ID        uuid.UUID
	MediaId   MediaID
	Monitored bool
	Status    ItemStatus
	Metadata  json.RawMessage
}

type MediaWithItems struct {
	Media Media
	Items []MediaItem
}

type MediaSummary struct {
	ID            MediaID
	Type          MediaType
	Title         string
	OriginalTitle string
	OriginalLang  string
	// Monitored Is any item monitored
	Monitored       bool
	Status          string
	Summary         string
	ReleaseDate     time.Time
	PrimaryIdentity MediaIdentity
	PosterPath      ImageURL
	// Metadata Preferably doesn't contain a lot of data,
	// it exists as a way for modules to add functionality
	Metadata any
}

type MediaMetadata json.RawMessage

func (m MediaMetadata) Raw() json.RawMessage {
	return json.RawMessage(m)
}

func (m MediaMetadata) Decode(v any) error {
	if len(m) == 0 {
		return nil
	}
	return sonic.Unmarshal(m, v)
}

func NewMetadata(v any) (MediaMetadata, error) {
	b, err := sonic.Marshal(v)
	if err != nil {
		return nil, err
	}
	return MediaMetadata(b), nil
}
