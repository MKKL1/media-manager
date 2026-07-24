package domain

type SearchQuery struct {
	Query    string
	Year     int
	Language string
}

type SearchResult struct {
	PrimaryIdentity MediaIdentity
	MediaType       MediaType
	Title           string
	Year            int
	Overview        string
	Poster          ImageURL
	Popularity      float64
}
