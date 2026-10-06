package view

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"

	"server/internal/library"
	"server/internal/plugin"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrUnknownFormat = errors.New("unknown format")
)

type NotFoundKind string

const (
	NotFoundWork        NotFoundKind = "work"
	NotFoundArrangement NotFoundKind = "arrangement"
	NotFoundType        NotFoundKind = "type"
)

type NotFound struct {
	Kind      NotFoundKind
	Name, Why string
}

func (e NotFound) Error() string        { return fmt.Sprintf("%s %s not found: %s", e.Kind, e.Name, e.Why) }
func (e NotFound) Is(target error) bool { return target == ErrNotFound }

type Type interface {
	Formats() []plugin.Tag
	FieldsRead() []library.Field

	Names(ctx context.Context, as plugin.Tag, work Entry) (map[library.ID]string, error)
}

type Entry struct {
	ID      library.ID        `json:"id"`
	Name    string            `json:"name,omitempty"`
	Values  map[string]string `json:"values,omitempty"`
	Entries []Entry           `json:"entries,omitempty"`
}

type Shown struct {
	Arrangement string     `json:"arrangement"`
	Type        plugin.Tag `json:"type,omitempty"`
	Work        Entry      `json:"work"`
}

type Views struct {
	Library *library.Library
	Types   func() []Type
}

type Arrangement string

const FetchedIn Arrangement = ""

const FirstTypedTag plugin.Tag = ""

func (v *Views) Show(ctx context.Context, work library.ID, in Arrangement, as plugin.Tag) (Shown, error) {
	viewed, top, ok := v.work(work)
	if !ok {
		return Shown{}, NotFound{NotFoundWork, work.String(), "not a work"}
	}
	if in != FetchedIn {
		provider, name, ok := strings.Cut(string(in), "/")
		if !ok {
			provider, name = string(work.Provider), string(in)
		}
		if viewed.Provider() != library.Provider(provider) || viewed.Name() != name {
			target, ok := v.arrangement(library.Provider(provider), name)
			if ok {
				top, ok = sameWork(top, target)
			}
			if !ok {
				return Shown{}, NotFound{NotFoundArrangement, provider + "/" + name, work.String() + " is not fetched there"}
			}
			viewed = target
		}
	}
	shown := Shown{Arrangement: string(viewed.Provider()) + "/" + viewed.Name()}

	var tags []plugin.Tag
	for _, tag := range strings.Fields(pick(top, plugin.TagsField, viewed)) {
		tags = append(tags, plugin.Tag(tag))
	}
	typ, as, err := v.typeFor(as, tags)
	if err != nil {
		return Shown{}, err
	}
	fields := []library.Field{plugin.TagsField}
	if typ != nil {
		fields = append(fields, typ.FieldsRead()...)
	}
	shown.Work = entryAsViewed(top, viewed, fields)
	if typ == nil {
		return shown, nil
	}
	names, err := typ.Names(ctx, as, shown.Work)
	if err != nil {
		return Shown{}, fmt.Errorf("name %s as %s: %w", work, as, err)
	}
	shown.Type = as
	setNames(&shown.Work, names)
	return shown, nil
}

func (v *Views) arrangement(p library.Provider, name string) (library.Arrangement, bool) {
	for a := range v.Library.Arrangements(p) {
		if a.Name() == name {
			return a, true
		}
	}
	return library.Arrangement{}, false
}

func (v *Views) work(id library.ID) (library.Arrangement, library.Entry, bool) {
	if !slices.ContainsFunc(slices.Collect(v.Library.Works()), func(w library.Entry) bool { return w.ID() == id }) {
		return library.Arrangement{}, library.Entry{}, false
	}
	for a := range v.Library.Arrangements(id.Provider) {
		for top := range a.Top() {
			if top.ID() == id {
				return a, top, true
			}
		}
	}
	return library.Arrangement{}, library.Entry{}, false
}

func sameWork(work library.Entry, target library.Arrangement) (library.Entry, bool) {
	held := map[library.ID]bool{}
	for e := range entriesUnder(work) {
		if item, ok := e.AsItem(); ok {
			for member := range item.Entries() {
				held[member.ID()] = true
			}
		}
	}
	for top := range target.Top() {
		for e := range entriesUnder(top) {
			if held[e.ID()] {
				return top, true
			}
		}
	}
	return library.Entry{}, false
}

func entriesUnder(e library.Entry) iter.Seq[library.Entry] {
	return func(yield func(library.Entry) bool) {
		var walk func(library.Entry) bool
		walk = func(e library.Entry) bool {
			if !yield(e) {
				return false
			}
			if g, ok := e.AsGroup(); ok {
				for child := range g.Entries() {
					if !walk(child) {
						return false
					}
				}
			}
			return true
		}
		walk(e)
	}
}

func (v *Views) typeFor(as plugin.Tag, tags []plugin.Tag) (Type, plugin.Tag, error) {
	find := func(format plugin.Tag) Type {
		for _, t := range v.Types() {
			if slices.Contains(t.Formats(), format) {
				return t
			}
		}
		return nil
	}
	if as != FirstTypedTag {
		if t := find(as); t != nil {
			return t, as, nil
		}
		return nil, "", NotFound{NotFoundType, string(as), "no metadata type formats it"}
	}
	for _, tag := range tags {
		if t := find(tag); t != nil {
			return t, tag, nil
		}
	}
	return nil, "", nil
}

func entryAsViewed(e library.Entry, a library.Arrangement, fields []library.Field) Entry {
	viewed := Entry{ID: e.ID(), Values: map[string]string{}}
	for _, f := range fields {
		if text := pick(e, f, a); text != "" {
			viewed.Values[string(f)] = text
		}
	}
	if g, ok := e.AsGroup(); ok {
		for child := range g.Entries() {
			viewed.Entries = append(viewed.Entries, entryAsViewed(child, a, fields))
		}
	}
	return viewed
}

func pick(e library.Entry, f library.Field, a library.Arrangement) string {
	other := ""
	for s, text := range e.Values(f) {
		switch s := s.(type) {
		case library.Value:
			return text
		case library.Fetched:
			if s.Provider == a.Provider() && s.Arrangement == a.Name() {
				return text
			}
		}
		if other == "" {
			other = text
		}
	}
	if other != "" {
		return other
	}
	if item, ok := e.AsItem(); ok {
		for member := range item.Entries() {
			for _, text := range member.Values(f) {
				return text
			}
		}
	}
	return ""
}

func setNames(e *Entry, names map[library.ID]string) {
	e.Name = names[e.ID]
	for i := range e.Entries {
		setNames(&e.Entries[i], names)
	}
}
