package domain

import (
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

type MediaType string

type MediaId uuid.UUID

func (id MediaId) UUID() uuid.UUID    { return uuid.UUID(id) }
func (id MediaId) String() string     { return uuid.UUID(id).String() }
func NewMediaID(id uuid.UUID) MediaId { return MediaId(id) }
func GenerateMediaID() MediaId        { return MediaId(uuid.New()) }
func (id MediaId) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(id).String())
}

func (id *MediaId) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	*id = MediaId(parsed)
	return nil
}

type ExternalId struct {
	Provider string `json:"provider"`
	Id       string `json:"id"`
}

func (id ExternalId) String() string               { return id.Provider + ":" + id.Id }
func NewExternalId(provider, id string) ExternalId { return ExternalId{provider, id} }

type ItemStatus string

type Media struct {
	ID                MediaId
	Type              MediaType
	Title             string
	Status            string
	Monitored         bool
	PrimaryExternalId ExternalId
	ExternalIds       []ExternalId
	Metadata          json.RawMessage
	CreatedAt         time.Time
	LastSync          time.Time
	UpdatedAt         time.Time
}

type MediaItem struct {
	ID        uuid.UUID
	MediaId   MediaId
	Monitored bool
	Status    ItemStatus
	Metadata  json.RawMessage
}

type MediaWithItems struct {
	Media Media
	Items []MediaItem
}

type MediaSummary struct {
	Id            MediaId   `json:"id"`
	Type          MediaType `json:"type"`
	Title         string    `json:"title"`
	OriginalTitle string    `json:"originalTitle"`
	OriginalLang  string    `json:"originalLang"`
	// Monitored Is any item monitored
	Monitored   bool       `json:"monitored"`
	Status      string     `json:"status"`
	Summary     string     `json:"summary"`
	ReleaseDate time.Time  `json:"releaseDate"`
	Source      ExternalId `json:"source"`
	PosterPath  string     `json:"posterPath"`
	// Metadata Preferably doesn't contain a lot of data,
	// it exists as a way for modules to add functionality
	Metadata any `json:"metadata"`
}
