package domain

import "strings"

type ProviderName string

const (
	ProviderTMDB     ProviderName = "tmdb"
	ProviderTVDB     ProviderName = "tvdb"
	ProviderIMDB     ProviderName = "imdb"
	ProviderAniDB    ProviderName = "anidb"
	ProviderMAL      ProviderName = "mal"
	ProviderAniList  ProviderName = "anilist"
	ProviderWikidata ProviderName = "wikidata"
)

type ProviderKind struct {
	ProviderName ProviderName

	//TODO bad name for that, also it shouldn't really be accessed, but it has to be exported as I need it for jobs
	Kind string
}

func NewProviderKind(source ProviderName, kind string) ProviderKind {
	return ProviderKind{ProviderName: source, Kind: kind}
}

//TODO SourceKindFromString should probably do more checks and return error

func SourceKindFromString(s string) ProviderKind {
	i := strings.IndexByte(s, ':')
	if i == -1 {
		return NewProviderKind(ProviderName(s), "")
	}
	return NewProviderKind(ProviderName(s[:i]), s[i+1:])
}

func (k ProviderKind) String() string {
	if k.Kind == "" {
		return string(k.ProviderName)
	}
	return string(k.ProviderName) + ":" + k.Kind
}

var (
	KindTMDBTV      = NewProviderKind(ProviderTMDB, "tv")
	KindTMDBMovie   = NewProviderKind(ProviderTMDB, "movie")
	KindTMDBSeason  = NewProviderKind(ProviderTMDB, "season")
	KindTMDBEpisode = NewProviderKind(ProviderTMDB, "episode")
	KindTMDBEpGroup = NewProviderKind(ProviderTMDB, "eg")
	KindTVDB        = NewProviderKind(ProviderTVDB, "")
	KindIMDB        = NewProviderKind(ProviderIMDB, "")
	KindAniDB       = NewProviderKind(ProviderAniDB, "")
	KindMAL         = NewProviderKind(ProviderMAL, "")
	KindAniList     = NewProviderKind(ProviderAniList, "")
	KindWikidata    = NewProviderKind(ProviderWikidata, "")
)

// TODO MediaIdentity can be confused for MediaID, but those are two completely different things

type MediaIdentity struct {
	Kind ProviderKind `json:"kind"`
	ID   string       `json:"id"`
}

func (r MediaIdentity) String() string {
	return r.Kind.String() + ":" + r.ID
}
func NewMediaIdentity(kind ProviderKind, id string) MediaIdentity {
	return MediaIdentity{kind, id}
}
