package tv

import (
	"server/internal/domain"
	"time"

	"github.com/google/uuid"
)

const MediaType domain.MediaType = "tv"

type Show struct {
	domain.MediaItem
	Title string
	Year  int
}

type Metadata struct {
	OriginalTitle string           `json:"originalTitle,omitempty"`
	Overview      string           `json:"overview,omitempty"`
	Tagline       string           `json:"tagline,omitempty"`
	FirstAirDate  time.Time        `json:"firstAirDate,omitempty"`
	LastAirDate   time.Time        `json:"lastAirDate,omitempty"`
	SeasonCount   int              `json:"seasonCount,omitempty"`
	EpisodeCount  int              `json:"episodeCount,omitempty"`
	Runtime       int              `json:"runtime,omitempty"`
	Genres        []string         `json:"genres,omitempty"`
	Poster        string           `json:"poster,omitempty"`
	Backdrop      string           `json:"backdrop,omitempty"`
	Seasons       []SeasonMetadata `json:"seasons,omitempty"`
}

type SeasonMetadata struct {
	SeasonNumber int           `json:"seasonNumber"`
	Episodes     []EpisodeInfo `json:"episodes"`
}

type EpisodeInfo struct {
	EpisodeNumber int       `json:"episodeNumber"`
	ID            uuid.UUID `json:"id"`
}

type EpisodeMetadata struct {
	OriginalTitle string    `json:"originalTitle,omitempty"`
	Overview      string    `json:"overview,omitempty"`
	AirDate       time.Time `json:"airDate,omitempty"`
	Runtime       int       `json:"runtime,omitempty"`
	Still         string    `json:"still,omitempty"`
	SeasonNumber  int       `json:"seasonNumber,omitempty"`
	EpisodeNumber int       `json:"episodeNumber,omitempty"`
}

type Episode struct {
	domain.MediaItem
	ShowID        domain.MediaId
	SeasonNumber  int
	EpisodeNumber int
	Title         string
	AirDate       time.Time
}

type SearchQuery struct {
	Title    string
	Year     int
	Language string
}

type SearchResult struct {
	ExternalID domain.ExternalId
	Title      string
	Year       int
	Overview   string
	Poster     string
	Popularity float64
}

type ProviderShow struct {
	ExternalID       domain.ExternalId
	ExternalIDs      []domain.ExternalId
	Title            string
	OriginalTitle    string
	OriginalLanguage string
	Overview         string
	Tagline          string
	Status           string
	ContentRating    string
	FirstAirDate     time.Time
	LastAirDate      time.Time
	Year             int
	Runtime          int
	SeasonCount      int
	EpisodeCount     int
	Genres           []string
	OriginCountry    []string
	Networks         []string
	CreatedBy        []string
	Poster           string
	Backdrop         string
	Rating           float32
	VoteCount        int
	Popularity       float64
}

type ProviderSeason struct {
	SeasonNumber int
	Rating       *float32
	VoteCount    *int
	ExternalID   *domain.ExternalId
	Title        *string
	Episodes     []ProviderEpisode
}

type ProviderEpisode struct {
	ExternalID     domain.ExternalId
	ShowExternalID domain.ExternalId
	SeasonNumber   int
	EpisodeNumber  int
	Title          string
	Overview       string
	AirDate        time.Time
	Runtime        int
	Still          string
	Rating         float32
	VoteCount      int
	IsSeasonFinale bool
}
