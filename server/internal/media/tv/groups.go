package tv

import "context"

//Mainly here because TMDB supports it. I don't know how to make it generic enough for other use cases.

// EpisodeGroupType mirrors TMDB's type codes
type EpisodeGroupType int

const (
	EpisodeGroupTypeOriginalAirDate EpisodeGroupType = 1
	EpisodeGroupTypeAbsolute        EpisodeGroupType = 2
	EpisodeGroupTypeDVD             EpisodeGroupType = 3
	EpisodeGroupTypeDigital         EpisodeGroupType = 4
	EpisodeGroupTypeStoryArc        EpisodeGroupType = 5
	EpisodeGroupTypeProduction      EpisodeGroupType = 6
	EpisodeGroupTypeTV              EpisodeGroupType = 7
)

type EpisodeGroup struct {
	ID           string
	Name         string
	Description  string
	Type         EpisodeGroupType
	EpisodeCount int
	GroupCount   int
}

type EpisodeGroupDetail struct {
	ID     string
	Name   string
	Type   EpisodeGroupType
	Groups []EpisodeGrouping
}

// EpisodeGrouping is one "season" or "arc" within a group.
type EpisodeGrouping struct {
	ID       string
	Name     string
	Order    int
	Episodes []ProviderEpisode
}

type EpisodeGroupFetcher interface {
	GetEpisodeGroups(ctx context.Context, showID string) ([]EpisodeGroup, error)
	GetEpisodeGroupDetail(ctx context.Context, groupID string) (*EpisodeGroupDetail, error)
}
