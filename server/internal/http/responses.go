package http

import (
	"time"

	"server/internal/domain"
)

// MediaPageResponse is a wrapper for paginated endpoints
type MediaPageResponse struct {
	Items  []MediaSummaryResponse `json:"items"`
	Total  int                    `json:"total"`
	Offset int                    `json:"offset"`
	Limit  int                    `json:"limit"`
}

// MediaIdentityResponse formats the domain.MediaIdentity
type MediaIdentityResponse struct {
	Provider string `json:"provider"`
	Kind     string `json:"kind,omitempty"`
	ID       string `json:"id"`
}

// MediaSummaryResponse corresponds to domain.MediaSummary
type MediaSummaryResponse struct {
	ID              string                `json:"id"`
	Type            string                `json:"type"`
	Title           string                `json:"title"`
	OriginalTitle   string                `json:"original_title"`
	OriginalLang    string                `json:"original_lang"`
	Monitored       bool                  `json:"monitored"`
	Status          string                `json:"status"`
	Summary         string                `json:"summary"`
	ReleaseDate     string                `json:"release_date"`
	PrimaryIdentity MediaIdentityResponse `json:"primary_identity"`
	PosterPath      string                `json:"poster_path"`
	Metadata        any                   `json:"metadata"`
}

// SearchResultResponse corresponds to domain.SearchResult
type SearchResultResponse struct {
	Identity   MediaIdentityResponse `json:"identity"`
	MediaType  string                `json:"media_type"`
	Title      string                `json:"title"`
	Year       int                   `json:"year"`
	Overview   string                `json:"overview"`
	Poster     string                `json:"poster"`
	Popularity float64               `json:"popularity"`
}

// PullMediaResponse corresponds to the PullMedia endpoint result
type PullMediaResponse struct {
	Status     string `json:"status"`
	WorkflowID string `json:"workflow_id"`
}

func toMediaIdentityResponse(id domain.MediaIdentity) MediaIdentityResponse {
	return MediaIdentityResponse{
		Provider: string(id.Kind.ProviderName),
		Kind:     id.Kind.Kind,
		ID:       id.ID,
	}
}

func toMediaSummaryResponse(summary domain.MediaSummary) MediaSummaryResponse {
	var releaseDate string
	if !summary.ReleaseDate.IsZero() {
		releaseDate = summary.ReleaseDate.Format(time.RFC3339)
	}

	return MediaSummaryResponse{
		ID:              summary.ID.String(),
		Type:            string(summary.Type),
		Title:           summary.Title,
		OriginalTitle:   summary.OriginalTitle,
		OriginalLang:    summary.OriginalLang,
		Monitored:       summary.Monitored,
		Status:          summary.Status,
		Summary:         summary.Summary,
		ReleaseDate:     releaseDate,
		PrimaryIdentity: toMediaIdentityResponse(summary.PrimaryIdentity),
		PosterPath:      string(summary.PosterPath),
		Metadata:        summary.Metadata,
	}
}

func toSearchResultResponse(res domain.SearchResult) SearchResultResponse {
	return SearchResultResponse{
		Identity:   toMediaIdentityResponse(res.PrimaryIdentity),
		MediaType:  string(res.MediaType),
		Title:      res.Title,
		Year:       res.Year,
		Overview:   res.Overview,
		Poster:     string(res.Poster),
		Popularity: res.Popularity,
	}
}
