package library

import (
	"cmp"
	"iter"
	"slices"
)

type Entry struct {
	library *Library
	pinned  *snapshot
	number  int32
}

func (m *snapshot) entry(n int32) Entry { return Entry{m.library, m, n} }

func (e Entry) snapshot() *snapshot {
	if e.pinned != nil {
		return e.pinned
	}
	return e.library.latest()
}

type Group struct{ entry Entry }

type Item struct {
	snapshot *snapshot
	number   int32
}

type Report struct {
	Kind    ReportKind
	Entries []Entry
}

type ReportKind string

const (
	Unplaced      ReportKind = "unplaced"
	Undecided     ReportKind = "undecided"
	Disagreement  ReportKind = "disagreement"
	Conflict      ReportKind = "conflict"
	UnappliedEdit ReportKind = "unapplied edit"
)

func (l *Library) Entry(id ID) (Entry, bool) {
	n, ok := l.entryNumbers.lookUp(id)
	if !ok || !l.latest().isNamed(n) {
		return Entry{}, false
	}
	return Entry{library: l, number: n}, true
}

func (l *Library) Works() iter.Seq[Entry] {
	m := l.latest()
	return func(yield func(Entry) bool) {
		for _, index := range m.works {
			if !yield(m.entry(m.topLevels[index].entries[0])) {
				return
			}
		}
	}
}

func (l *Library) Reports() iter.Seq[Report] {
	m := l.latest()
	return func(yield func(Report) bool) {
		for _, item := range m.unplaced {
			if !yield(Report{Unplaced, slices.Collect(Item{m, item}.Entries())}) {
				return
			}
		}
		for _, n := range m.undecided {
			if !yield(Report{Undecided, []Entry{m.entry(n)}}) {
				return
			}
		}
		for _, p := range m.disagreements {
			if !yield(Report{Disagreement, []Entry{m.entry(p.lower), m.entry(p.higher)}}) {
				return
			}
		}
		for _, p := range m.conflicts {
			if !yield(Report{Conflict, []Entry{m.entry(p.lower), m.entry(p.higher)}}) {
				return
			}
		}
		for _, n := range m.unappliedEdits {
			if !yield(Report{UnappliedEdit, []Entry{m.entry(n)}}) {
				return
			}
		}
	}
}

func (e Entry) ID() ID {
	if e.pinned != nil && int(e.number) < len(e.pinned.ids) {
		return e.pinned.ids[e.number]
	}
	return e.library.entryNumbers.id(e.number)
}

func (e Entry) String() string { return e.ID().String() }

func (e Entry) AsGroup() (Group, bool) {
	m := e.snapshot()
	return Group{m.entry(e.number)}, m.isGroup(e.number)
}

func (e Entry) AsItem() (Item, bool) {
	m := e.snapshot()
	if int(e.number) >= m.count() || m.itemOf[e.number] == none {
		return Item{}, false
	}
	return Item{m, m.itemOf[e.number]}, true
}

func (e Entry) Values(f Field) iter.Seq2[Statement, string] {
	m := e.snapshot()
	return func(yield func(Statement, string) bool) {
		field, ok := m.fieldNumbers[f]
		if !ok {
			return
		}
		id := e.ID()
		if text, ok := m.valueEdits[entryAndField{e.number, field}]; ok {
			if !yield(Value{Entry: id, Field: f, Text: text}, text) {
				return
			}
		}
		for _, p := range m.placesOf(e.number) {
			t := m.topLevels[p.topLevelIndex]
			if text, ok := t.value(p.indexInTopLevel, field); ok {
				if !yield(m.fetchedKey(t), text) {
					return
				}
			}
		}
	}
}

func (m *snapshot) fetchedKey(t *topLevel) Fetched {
	name := m.arrangementNames[t.arrangement]
	return Fetched{Provider: name.provider, Arrangement: name.name, Top: m.ids[t.entries[0]].ProviderID}
}

func (e Entry) Statements() iter.Seq[Statement] {
	m := e.snapshot()
	return func(yield func(Statement) bool) {
		id := e.ID()
		for _, p := range m.placesOf(e.number) {
			if !yield(m.fetchedKey(m.topLevels[p.topLevelIndex])) {
				return
			}
		}
		for _, k := range m.linksOf(e.number) {
			link := linkSeenFrom(id, m.ids[k.other], k.kindSeenFromThisEntry, m.linkWhy[k.link])
			var s Statement = link
			if k.mapping != none {
				mapping := m.mappings[k.mapping]
				s = Mapping{Provider: mapping.provider, OwnID: mapping.ownID, Links: []Link{link}}
			}
			if !yield(s) {
				return
			}
		}
		var fields []Field
		for f, number := range m.fieldNumbers {
			if _, ok := m.valueEdits[entryAndField{e.number, number}]; ok {
				fields = append(fields, f)
			}
		}
		slices.Sort(fields)
		for _, f := range fields {
			if !yield(Value{Entry: id, Field: f, Text: m.valueEdits[entryAndField{e.number, m.fieldNumbers[f]}]}) {
				return
			}
		}
		moves := slices.Clone(m.moves[e.number])
		slices.SortFunc(moves, func(x, y addToGroup) int { return cmp.Compare(x.editOrder, y.editOrder) })
		for _, mv := range moves {
			if !yield(Move{Entry: id, Group: m.ids[mv.group]}) {
				return
			}
		}
		if chosen, decided := m.choices[e.number]; decided {
			yield(Choice{Entry: id, Chosen: chosen})
		}
	}
}

func (g Group) Entry() Entry { return g.entry }

func (g Group) Entries() iter.Seq[Entry] {
	m := g.entry.snapshot()
	return func(yield func(Entry) bool) {
		places := m.placesOf(g.entry.number)
		if len(places) == 0 {
			return
		}
		for n := range m.entriesDirectlyUnder(m.topLevels[places[0].topLevelIndex], places[0].indexInTopLevel) {
			if !yield(m.entry(n)) {
				return
			}
		}
	}
}

func (i Item) Entries() iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		for _, n := range i.snapshot.membersOf(i.number) {
			if !yield(i.snapshot.entry(n)) {
				return
			}
		}
	}
}

func (l *Library) latest() *snapshot {
	if l.needsNewSnapshot.Load() && !l.loadingFromStore.Load() {
		l.writeLock.Lock()
		if l.needsNewSnapshot.Load() {
			l.needsNewSnapshot.Store(false)
			l.latestSnapshot.Store(l.newSnapshot())
		}
		l.writeLock.Unlock()
	}
	return l.latestSnapshot.Load()
}

func (l *Library) snapshotWhileWriting() *snapshot {
	if l.needsNewSnapshot.Load() {
		l.needsNewSnapshot.Store(false)
		l.latestSnapshot.Store(l.newSnapshot())
	}
	return l.latestSnapshot.Load()
}
