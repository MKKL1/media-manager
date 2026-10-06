package main

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TVDetails struct {
	ID               int             `json:"id"`
	Name             string          `json:"name"`
	OriginalName     string          `json:"original_name"`
	OriginalLanguage string          `json:"original_language"`
	OriginCountry    []string        `json:"origin_country"`
	Overview         string          `json:"overview"`
	Tagline          string          `json:"tagline"`
	Status           string          `json:"status"`
	Homepage         string          `json:"homepage"`
	InProduction     bool            `json:"in_production"`
	FirstAirDate     string          `json:"first_air_date"`
	LastAirDate      string          `json:"last_air_date"`
	PosterPath       string          `json:"poster_path"`
	BackdropPath     string          `json:"backdrop_path"`
	Popularity       float64         `json:"popularity"`
	VoteAverage      float64         `json:"vote_average"`
	VoteCount        int             `json:"vote_count"`
	NumberOfSeasons  int             `json:"number_of_seasons"`
	NumberOfEpisodes int             `json:"number_of_episodes"`
	EpisodeRunTime   []int           `json:"episode_run_time"`
	Genres           []Genre         `json:"genres"`
	Seasons          []SeasonSummary `json:"seasons"`

	ExternalIDs   *ExternalIDs         `json:"external_ids,omitempty"`
	EpisodeGroups *EpisodeGroupsResult `json:"episode_groups,omitempty"`
}

type MovieDetails struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	OriginalTitle    string  `json:"original_title"`
	OriginalLanguage string  `json:"original_language"`
	Overview         string  `json:"overview"`
	Tagline          string  `json:"tagline"`
	Status           string  `json:"status"`
	Homepage         string  `json:"homepage"`
	Adult            bool    `json:"adult"`
	Video            bool    `json:"video"`
	ReleaseDate      string  `json:"release_date"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	Runtime          int     `json:"runtime"`
	Budget           int64   `json:"budget"`
	Revenue          int64   `json:"revenue"`
	Popularity       float64 `json:"popularity"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	IMDbID           string  `json:"imdb_id"`
	Genres           []Genre `json:"genres"`

	ExternalIDs *ExternalIDs `json:"external_ids,omitempty"`
}

type SeasonSummary struct {
	ID           int    `json:"id"`
	SeasonNumber int    `json:"season_number"`
	EpisodeCount int    `json:"episode_count"`
	Name         string `json:"name"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
}

type MultiSearchResult struct {
	ID               int      `json:"id"`
	MediaType        string   `json:"media_type"`
	Title            string   `json:"title"`
	Name             string   `json:"name"`
	OriginalTitle    string   `json:"original_title"`
	OriginalName     string   `json:"original_name"`
	OriginalLanguage string   `json:"original_language"`
	Overview         string   `json:"overview"`
	PosterPath       string   `json:"poster_path"`
	BackdropPath     string   `json:"backdrop_path"`
	ReleaseDate      string   `json:"release_date"`
	FirstAirDate     string   `json:"first_air_date"`
	Popularity       float64  `json:"popularity"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
	GenreIDs         []int    `json:"genre_ids"`
	OriginCountry    []string `json:"origin_country"`
	Adult            bool     `json:"adult"`
}

type Season struct {
	ID           int       `json:"id"`
	AirDate      string    `json:"air_date"`
	EpisodeCount int       `json:"episode_count"`
	Name         string    `json:"name"`
	Overview     string    `json:"overview"`
	PosterPath   string    `json:"poster_path"`
	SeasonNumber int       `json:"season_number"`
	Episodes     []Episode `json:"episodes"`
}

type Episode struct {
	ID             int     `json:"id"`
	AirDate        string  `json:"air_date"`
	EpisodeNumber  int     `json:"episode_number"`
	Name           string  `json:"name"`
	Overview       string  `json:"overview"`
	ProductionCode string  `json:"production_code"`
	Runtime        int     `json:"runtime"`
	SeasonNumber   int     `json:"season_number"`
	ShowID         int     `json:"show_id"`
	StillPath      string  `json:"still_path"`
	VoteAverage    float64 `json:"vote_average"`
	VoteCount      int     `json:"vote_count"`
	EpisodeType    string  `json:"episode_type"`
}

type ExternalIDs struct {
	ID          int    `json:"id"`
	IMDbID      string `json:"imdb_id"`
	FreebaseMID string `json:"freebase_mid"`
	FreebaseID  string `json:"freebase_id"`
	TVDBID      int    `json:"tvdb_id"`
	TvrageID    int    `json:"tvrage_id"`
	WikidataID  string `json:"wikidata_id"`
	FacebookID  string `json:"facebook_id"`
	InstagramID string `json:"instagram_id"`
	TwitterID   string `json:"twitter_id"`
}

type EpisodeGroupsResult struct {
	Results []EpisodeGroupSummary `json:"results"`
}

type EpisodeGroupSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         int    `json:"type"`
	EpisodeCount int    `json:"episode_count"`
	GroupCount   int    `json:"group_count"`
}

type EpisodeGroupDetailResponse struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Description  string                  `json:"description"`
	Type         int                     `json:"type"`
	EpisodeCount int                     `json:"episode_count"`
	Groups       []EpisodeGroupingDetail `json:"groups"`
}

type EpisodeGroupingDetail struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Order    int       `json:"order"`
	Episodes []Episode `json:"episodes"`
}
