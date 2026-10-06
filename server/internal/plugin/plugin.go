package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"server/internal/library"
)

var (
	ErrNotFound          = errors.New("not found at the provider")
	ErrNoSuchArrangement = errors.New("arrangement not offered")
	ErrNotLoaded         = errors.New("plugin did not load")
)

type Plugin interface {
	Provider() library.Provider

	Fetch(ctx context.Context, arrangement, ref string) (library.Tree, error)
}

type Arrangements interface {
	Arrangements() []string
}

const Default = "default"

func ArrangementsOf(p Plugin) []string {
	if a, ok := p.(Arrangements); ok {
		return a.Arrangements()
	}
	return []string{Default}
}

type Tags interface {
	Tags() []Tag
}

type Tag string

const TagsField library.Field = "tags"

func TagsOf(p Plugin) []Tag {
	if t, ok := p.(Tags); ok {
		return t.Tags()
	}
	return nil
}

type Searcher interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

type SearchResult struct {
	Ref    string
	Values map[library.Field]string
}

func Search(ctx context.Context, p Plugin, query string) ([]SearchResult, error) {
	s, ok := p.(Searcher)
	if !ok {
		return nil, fmt.Errorf("%s can't search: %w", p.Provider(), errors.ErrUnsupported)
	}
	found, err := s.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: search %q: %w", p.Provider(), query, err)
	}
	return found, nil
}

func Fetch(ctx context.Context, lib *library.Library, p Plugin, arrangement, ref string) (library.ID, error) {
	arrangement, err := offeredOrDefault(p, arrangement)
	if err != nil {
		return library.ID{}, err
	}
	top, err := p.Fetch(ctx, arrangement, ref)
	if err != nil {
		return library.ID{}, fmt.Errorf("%s: fetch %q in %s: %w", p.Provider(), ref, arrangement, err)
	}
	if err := lib.Put(ctx, fetched(p, arrangement, top)); err != nil {
		return library.ID{}, err
	}
	return library.ID{Provider: p.Provider(), ProviderID: top.ProviderID}, nil
}

func fetched(p Plugin, arrangement string, top library.Tree) library.Fetched {
	return library.Fetched{
		Provider: p.Provider(), Arrangement: arrangement, Top: top.ProviderID,
		CanHoldEntries: top.CanHoldEntries, Values: top.Values, Entries: top.Entries,
	}
}

func Refresh(ctx context.Context, lib *library.Library, p Plugin, arrangement string) error {
	arrangement, err := offeredOrDefault(p, arrangement)
	if err != nil {
		return err
	}
	var tops []library.Entry
	for a := range lib.Arrangements(p.Provider()) {
		if a.Name() == arrangement {
			tops = slices.Collect(a.Top())
		}
	}
	var errs []error
	for _, top := range tops {
		id := top.ID()
		tree, err := p.Fetch(ctx, arrangement, id.ProviderID)
		switch {
		case errors.Is(err, ErrNotFound) || errors.Is(err, ErrNoSuchArrangement):
			_, err = lib.Remove(ctx, library.Fetched{Provider: p.Provider(), Arrangement: arrangement, Top: id.ProviderID})
		case err == nil:
			err = lib.Put(ctx, fetched(p, arrangement, tree))
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: refresh %v in %s: %w", p.Provider(), id, arrangement, err))
		}
	}
	return errors.Join(errs...)
}

func offeredOrDefault(p Plugin, arrangement string) (string, error) {
	offered := ArrangementsOf(p)
	if arrangement == "" && len(offered) > 0 {
		return offered[0], nil
	}
	if !slices.Contains(offered, arrangement) {
		return "", fmt.Errorf("%s %q: %w", p.Provider(), arrangement, ErrNoSuchArrangement)
	}
	return arrangement, nil
}

type Mappings interface {
	Provider() library.Provider

	FetchMapping(ctx context.Context, lib *library.Library, ref string) (library.Mapping, error)
}

func FetchMapping(ctx context.Context, lib *library.Library, p Mappings, ref string) error {
	mapping, err := p.FetchMapping(ctx, lib, ref)
	if err != nil {
		return fmt.Errorf("%s: fetch mapping %q: %w", p.Provider(), ref, err)
	}
	return lib.Put(ctx, mapping)
}
