package library

import (
	"context"

	"server/internal/event"
)

const (
	FetchedUpdated = "fetched.updated"
	FetchedDeleted = "fetched.deleted"
	MappingUpdated = "mapping.updated"
	MappingDeleted = "mapping.deleted"
	EditUpdated    = "edit.updated"
	EditDeleted    = "edit.deleted"
)

func EntrySubject(id ID) string { return "entries/" + id.String() }

type FetchedData struct {
	Arrangement string `json:"arrangement"`
}

type MappingData struct {
	OwnID string `json:"own_id"`
	Links int    `json:"links,omitempty"`
}

type EditData struct {
	Kind   string   `json:"kind"`
	Field  string   `json:"field,omitempty"`
	Value  string   `json:"value,omitempty"`
	Group  string   `json:"group,omitempty"`
	Other  string   `json:"other,omitempty"`
	Link   string   `json:"link,omitempty"`
	Why    []string `json:"why,omitempty"`
	Chosen *bool    `json:"chosen,omitempty"`
}

func (l *Library) publish(ctx context.Context, s Statement, deleted bool) {
	pick := func(updated, removed string) string {
		if deleted {
			return removed
		}
		return updated
	}
	e := event.Event{Type: pick(EditUpdated, EditDeleted)}
	switch s := s.(type) {
	case Fetched:
		e = event.Event{Type: pick(FetchedUpdated, FetchedDeleted), Subject: EntrySubject(ID{Provider: s.Provider, ProviderID: s.Top}), Data: FetchedData{s.Arrangement}}
	case Mapping:
		e = event.Event{Type: pick(MappingUpdated, MappingDeleted), Subject: "providers/" + string(s.Provider), Data: MappingData{s.OwnID, len(s.Links)}}
	case Value:
		e.Subject, e.Data = EntrySubject(s.Entry), EditData{Kind: "value", Field: string(s.Field), Value: s.Text}
	case Move:
		e.Subject, e.Data = EntrySubject(s.Entry), EditData{Kind: "move", Group: s.Group.String()}
	case Link:
		d := EditData{Kind: "link", Other: s.To.String()}
		if !deleted {
			d.Link = s.Kind.String()
		}
		for _, w := range s.Why {
			d.Why = append(d.Why, string(w))
		}
		e.Subject, e.Data = EntrySubject(s.From), d
	case Choice:
		d := EditData{Kind: "choice"}
		if !deleted {
			d.Chosen = &s.Chosen
		}
		e.Subject, e.Data = EntrySubject(s.Entry), d
	}
	l.events.Publish(ctx, e)
}
