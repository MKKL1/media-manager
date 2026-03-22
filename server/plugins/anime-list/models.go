package anime_list

import "encoding/xml"

type xmlAnimeList struct {
	XMLName xml.Name   `xml:"anime-list"`
	Anime   []xmlAnime `xml:"anime"`
}

type xmlAnime struct {
	AniDBID           int             `xml:"anidbid,attr"`
	TVDBID            string          `xml:"tvdbid,attr"`
	DefaultTVDBSeason string          `xml:"defaulttvdbseason,attr"`
	TMDBTv            string          `xml:"tmdbtv,attr"`
	TMDBSeason        string          `xml:"tmdbseason,attr"`
	TMDBId            string          `xml:"tmdbid,attr"`
	IMDBId            string          `xml:"imdbid,attr"`
	EpisodeOffset     int             `xml:"episodeoffset,attr"`
	Name              string          `xml:"name"`
	MappingList       *xmlMappingList `xml:"mapping-list"`
}

type xmlMappingList struct {
	Mappings []xmlMapping `xml:"mapping"`
}

type xmlMapping struct {
	AniDBSeason int    `xml:"anidbseason,attr"`
	TVDBSeason  int    `xml:"tvdbseason,attr"`
	Start       int    `xml:"start,attr"`
	End         int    `xml:"end,attr"`
	Offset      int    `xml:"offset,attr"`
	Content     string `xml:",chardata"` // ";1-4;2-5;"
}
