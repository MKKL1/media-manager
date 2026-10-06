package tvdb

type SeasonType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Season struct {
	ID     int        `json:"id"`
	Number int        `json:"number"`
	Name   string     `json:"name"`
	Type   SeasonType `json:"type"`
}

type Genre struct {
	Name string `json:"name"`
}

type RemoteID struct {
	ID         string `json:"id"`
	SourceName string `json:"sourceName"`
}

type Status struct {
	Name string `json:"name"`
}

type Series struct {
	ID               int        `json:"id"`
	Name             string     `json:"name"`
	Overview         string     `json:"overview"`
	OriginalLanguage string     `json:"originalLanguage"`
	FirstAired       string     `json:"firstAired"`
	Status           Status     `json:"status"`
	Genres           []Genre    `json:"genres"`
	RemoteIDs        []RemoteID `json:"remoteIds"`
	Seasons          []Season   `json:"seasons"`
}

type Release struct {
	Date string `json:"date"`
}

type Movie struct {
	ID               int        `json:"id"`
	Name             string     `json:"name"`
	Overview         string     `json:"overview"`
	OriginalLanguage string     `json:"originalLanguage"`
	Year             string     `json:"year"`
	FirstRelease     Release    `json:"first_release"`
	Genres           []Genre    `json:"genres"`
	RemoteIDs        []RemoteID `json:"remoteIds"`
}

type Translation struct {
	Name     string `json:"name"`
	Overview string `json:"overview"`
}

type Episode struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Overview       string `json:"overview"`
	Aired          string `json:"aired"`
	Number         int    `json:"number"`
	SeasonNumber   int    `json:"seasonNumber"`
	AbsoluteNumber int    `json:"absoluteNumber"`
}
