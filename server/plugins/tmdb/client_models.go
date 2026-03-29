package tmdb

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Network struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

type ProductionCompany struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

type Creator struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ProfilePath string `json:"profile_path"`
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
	Networks         []Network       `json:"networks"`
	CreatedBy        []Creator       `json:"created_by"`
	Seasons          []SeasonSummary `json:"seasons"`
}

type SeasonSummary struct {
	ID           int    `json:"id"`
	SeasonNumber int    `json:"season_number"`
	EpisodeCount int    `json:"episode_count"`
	Name         string `json:"name"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
}

type MovieDetails struct {
	ID                  int                 `json:"id"`
	Title               string              `json:"title"`
	OriginalTitle       string              `json:"original_title"`
	OriginalLanguage    string              `json:"original_language"`
	Overview            string              `json:"overview"`
	Tagline             string              `json:"tagline"`
	Status              string              `json:"status"`
	Homepage            string              `json:"homepage"`
	Adult               bool                `json:"adult"`
	Video               bool                `json:"video"`
	ReleaseDate         string              `json:"release_date"`
	PosterPath          string              `json:"poster_path"`
	BackdropPath        string              `json:"backdrop_path"`
	Runtime             int                 `json:"runtime"`
	Budget              int64               `json:"budget"`
	Revenue             int64               `json:"revenue"`
	Popularity          float64             `json:"popularity"`
	VoteAverage         float64             `json:"vote_average"`
	VoteCount           int                 `json:"vote_count"`
	IMDbID              string              `json:"imdb_id"`
	Genres              []Genre             `json:"genres"`
	ProductionCompanies []ProductionCompany `json:"production_companies"`
}

type TVShow struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	OriginalName     string   `json:"original_name"`
	OriginalLanguage string   `json:"original_language"`
	OriginCountry    []string `json:"origin_country"`
	Overview         string   `json:"overview"`
	FirstAirDate     string   `json:"first_air_date"`
	PosterPath       string   `json:"poster_path"`
	BackdropPath     string   `json:"backdrop_path"`
	Popularity       float64  `json:"popularity"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
}

type Movie struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	OriginalTitle    string  `json:"original_title"`
	OriginalLanguage string  `json:"original_language"`
	Overview         string  `json:"overview"`
	ReleaseDate      string  `json:"release_date"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	Adult            bool    `json:"adult"`
	Video            bool    `json:"video"`
	Popularity       float64 `json:"popularity"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
}

type SearchTVParams struct {
	Query string // required
	Page  int
	Year  int
}

type SearchMovieParams struct {
	Query string // required
	Page  int
	Year  int
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
