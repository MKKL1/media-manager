package library

import "slices"

func (l *Library) putMapping(s Mapping) (check func() error, apply func()) {
	name := mappingName{s.Provider, s.OwnID}
	return func() error {
			if s.Provider == "" || s.OwnID == "" {
				return refused("a mapping from %q under %q has no provider or own id", s.Provider, s.OwnID)
			}
			seen := map[pair]bool{}
			for _, k := range s.Links {
				if err := checkLinkKind(k.Kind); err != nil {
					return err
				}
				if err := l.checkLinkEnds(k); err != nil {
					return err
				}
				p, _ := l.pairOf(k)
				if seen[p] {
					return refused("%v and %v are linked twice under %v %q", k.From, k.To, s.Provider, s.OwnID)
				}
				seen[p] = true
			}
			return nil
		},
		func() {
			kept := &mapping{provider: s.Provider, ownID: s.OwnID, links: make([]providerLink, len(s.Links))}
			for i, k := range s.Links {
				p, kind := l.pairOf(k)
				kept.links[i] = providerLink{p, kind, slices.Clone(k.Why)}
			}
			if index, ok := l.mappingIndex[name]; ok {
				l.mappings[index] = kept
				return
			}
			l.mappingIndex[name] = int32(len(l.mappings))
			l.mappings = append(l.mappings, kept)
		}
}

func (l *Library) removeMapping(s Mapping) (check func() error, apply func()) {
	name := mappingName{s.Provider, s.OwnID}
	return func() error {
			if _, ok := l.mappingIndex[name]; !ok {
				return errNothingToDo
			}
			return nil
		},
		func() {
			if index, ok := l.mappingIndex[name]; ok {
				l.mappings[index] = nil
				delete(l.mappingIndex, name)
			}
		}
}

type mappingName struct {
	provider Provider
	ownID    string
}

type mapping struct {
	provider Provider
	ownID    string
	links    []providerLink
}

type providerLink struct {
	pair
	kind LinkKind
	why  []Evidence
}
