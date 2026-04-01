package movie

import (
	"server/internal/domain"
	"time"
)

const MediaType domain.MediaType = "movie"

type MovieResult struct {
	Title       string
	ExternalIDs []domain.MediaIdentity

	OriginalTitle    *string
	OriginalLanguage *string
	Overview         *string
	Tagline          *string
	Status           *string
	ReleaseDate      *time.Time
	Runtime          *int

	Genres []string
	Tags   []string
	Images []domain.ProviderImage
}

type Metadata struct {
	OriginalTitle    *string    `json:"original_title,omitempty"`
	OriginalLanguage *string    `json:"original_language,omitempty"`
	Overview         *string    `json:"overview,omitempty"`
	Tagline          *string    `json:"tagline,omitempty"`
	ReleaseDate      *time.Time `json:"release_date,omitempty"`
	Runtime          *int       `json:"runtime,omitempty"`
	Genres           []string   `json:"genres,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
}
