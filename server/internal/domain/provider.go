package domain

import "strings"

type Source string

const (
	SourceTMDB     Source = "tmdb"
	SourceTVDB     Source = "tvdb"
	SourceIMDB     Source = "imdb"
	SourceAniDB    Source = "anidb"
	SourceMAL      Source = "mal"
	SourceAniList  Source = "anilist"
	SourceWikidata Source = "wikidata"
)

type SourceKind string

const (
	KindTMDBTV      SourceKind = "tmdb:tv"
	KindTMDBMovie   SourceKind = "tmdb:movie"
	KindTMDBSeason  SourceKind = "tmdb:season"
	KindTMDBEpisode SourceKind = "tmdb:episode"
	KindTMDBEpGroup SourceKind = "tmdb:eg"
	KindTVDB        SourceKind = "tvdb"
	KindIMDB        SourceKind = "imdb"
	KindAniDB       SourceKind = "anidb"
	KindMAL         SourceKind = "mal"
	KindAniList     SourceKind = "anilist"
	KindWikidata    SourceKind = "wikidata"
)

func (k SourceKind) Source() Source {
	if i := strings.IndexByte(string(k), ':'); i != -1 {
		return Source(k[:i])
	}
	return Source(k)
}

// TODO MediaIdentity can be confused for MediaID, but those are two completely different things

type MediaIdentity struct {
	Kind SourceKind `json:"kind"`
	ID   string     `json:"id"`
}

func (r MediaIdentity) String() string {
	return string(r.Kind) + ":" + r.ID
}
func NewMediaIdentity(kind SourceKind, id string) MediaIdentity {
	return MediaIdentity{kind, id}
}
