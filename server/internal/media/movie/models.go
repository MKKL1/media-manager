package movie

import (
	"server/internal/domain"
	"time"
)

const MediaType domain.MediaType = "movie"

type Movie struct {
	domain.MediaItem
	Title string
	Year  int
}

type Metadata struct {
	OriginalTitle string    `json:"originalTitle,omitempty"`
	Overview      string    `json:"overview,omitempty"`
	Tagline       string    `json:"tagline,omitempty"`
	ReleaseDate   time.Time `json:"releaseDate,omitempty"`
	Runtime       int       `json:"runtime,omitempty"`
	Genres        []string  `json:"genres,omitempty"`
	Poster        string    `json:"poster,omitempty"`
	Backdrop      string    `json:"backdrop,omitempty"`
}

type SearchQuery struct {
	Title    string
	Year     int
	Language string
}

type SearchResult struct {
	ExternalID domain.MediaIdentity
	Title      string
	Year       int
	Overview   string
	Poster     string
	Popularity float64
}

type ProviderMovie struct {
	ExternalID  domain.MediaIdentity
	ExternalIDs []domain.MediaIdentity

	Title            string
	OriginalTitle    string
	OriginalLanguage string
	Overview         string
	Tagline          string
	Status           string
	ContentRating    string

	ReleaseDate time.Time
	Year        int
	Runtime     int

	Genres        []string
	OriginCountry []string

	Poster   string
	Backdrop string

	Rating     float32
	VoteCount  int
	Popularity float64

	Budget  int64
	Revenue int64
}
