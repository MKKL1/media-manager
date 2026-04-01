package tv

import (
	"server/internal/domain"
	"time"
)

const MediaType domain.MediaType = "tv"

type ShowResult struct {
	Title       string
	ExternalIDs []domain.MediaIdentity
	Seasons     []SeasonResult

	OriginalTitle    *string
	OriginalLanguage *string
	Overview         *string
	Tagline          *string
	Status           *string
	FirstAirDate     *time.Time
	LastAirDate      *time.Time
	Runtime          *int
	SeasonCount      *int
	EpisodeCount     *int

	Genres []string
	Tags   []string
	Images []domain.ProviderImage
}

// SeasonResult represents one season from a provider.
type SeasonResult struct {
	Number   int
	Title    *string
	Episodes []EpisodeResult
}

// EpisodeResult represents one episode from a provider.
type EpisodeResult struct {
	ExternalID    domain.MediaIdentity
	SeasonNumber  int
	EpisodeNumber int
	Title         string

	Overview *string
	AirDate  *time.Time
	Runtime  *int
	Still    *string
	IsFinale bool
}

// Metadata is the TV-specific metadata stored as a JSON blob.
type Metadata struct {
	OriginalTitle    *string          `json:"original_title,omitempty"`
	OriginalLanguage *string          `json:"original_language,omitempty"`
	Overview         *string          `json:"overview,omitempty"`
	Tagline          *string          `json:"tagline,omitempty"`
	FirstAirDate     *time.Time       `json:"first_air_date,omitempty"`
	LastAirDate      *time.Time       `json:"last_air_date,omitempty"`
	SeasonCount      *int             `json:"season_count,omitempty"`
	EpisodeCount     *int             `json:"episode_count,omitempty"`
	Runtime          *int             `json:"runtime,omitempty"`
	Genres           []string         `json:"genres,omitempty"`
	Tags             []string         `json:"tags,omitempty"`
	Seasons          []SeasonMetadata `json:"seasons,omitempty"`
}

// SeasonMetadata is stored per season inside Metadata.
type SeasonMetadata struct {
	SeasonNumber         int `json:"season_number"`
	EpisodeCount         int `json:"ep_count"`
	EpisodeReleasedCount int `json:"ep_released_count"`
}

// EpisodeMetadata is stored per media-item for individual episodes.
type EpisodeMetadata struct {
	OriginalTitle *string    `json:"original_title,omitempty"`
	Overview      *string    `json:"overview,omitempty"`
	AirDate       *time.Time `json:"air_date,omitempty"`
	Runtime       *int       `json:"runtime,omitempty"`
	Still         *string    `json:"still,omitempty"`
	SeasonNumber  int        `json:"season_number,omitempty"`
	EpisodeNumber int        `json:"episode_number,omitempty"`
}
