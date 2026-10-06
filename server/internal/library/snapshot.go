package library

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

type snapshot struct {
	library          *Library
	ids              []ID
	arrangementNames []arrangementName
	fieldNumbers     map[Field]uint16
	topLevels        []*topLevel
	mappings         []*mapping

	placesStart []int32
	places      []position

	linksStart []int32
	links      []linkEnd
	linkWhy    [][]Evidence

	itemOf       []int32
	membersStart []int32
	members      []int32

	valueEdits        map[entryAndField]string
	choices           map[int32]bool
	entriesAddedUnder map[int32][]int32
	addedAway         map[entryInArrangement]struct{}
	groupNowUnder     map[int32]int32

	works          []int32
	unplaced       []int32
	undecided      []int32
	disagreements  []pair
	conflicts      []pair
	unappliedEdits []int32
	moves          map[int32][]addToGroup
	named          []bool
}

type position struct{ topLevelIndex, indexInTopLevel int32 }

type linkEnd struct {
	other                 int32
	mapping               int32
	link                  int32
	kindSeenFromThisEntry LinkKind
	countsForItems        bool
}

func (l *Library) newSnapshot() *snapshot {
	m := &snapshot{
		library:          l,
		ids:              l.entryNumbers.all(),
		arrangementNames: l.arrangementNames,
		fieldNumbers:     l.fieldNumbers,
		topLevels:        slices.Clone(l.topLevels),
		mappings:         slices.Clone(l.mappings),
		valueEdits:       maps.Clone(l.valueEdits),
		choices:          maps.Clone(l.choices),
		moves:            make(map[int32][]addToGroup, len(l.addsToGroups)),
	}
	for entry, moves := range l.addsToGroups {
		m.moves[entry] = slices.Clone(moves)
	}
	m.findPlaces()
	m.buildLinks(l.links)
	m.buildItems()
	m.applyAdds(m.moves)
	m.findUndecidedAndConflicts()
	m.decideWorks()
	m.listUnappliedEdits()
	m.findNamed()
	return m
}

func (m *snapshot) findNamed() {
	m.named = make([]bool, m.count())
	for n := range int32(m.count()) {
		m.named[n] = m.isFetched(n) || len(m.linksOf(n)) > 0
	}
	for key := range m.valueEdits {
		m.named[key.entry] = true
	}
	for n := range m.choices {
		m.named[n] = true
	}
	for entry, moves := range m.moves {
		m.named[entry] = true
		for _, mv := range moves {
			m.named[mv.group] = true
		}
	}
}

func (m *snapshot) isNamed(n int32) bool { return int(n) < len(m.named) && m.named[n] }

func (m *snapshot) count() int { return len(m.ids) }

func (m *snapshot) placesOf(n int32) []position {
	if int(n) >= m.count() {
		return nil
	}
	return m.places[m.placesStart[n]:m.placesStart[n+1]]
}

func (m *snapshot) isFetched(n int32) bool { return len(m.placesOf(n)) > 0 }

func (m *snapshot) isGroup(n int32) bool {
	places := m.placesOf(n)
	return len(places) > 0 && m.topLevels[places[0].topLevelIndex].canHoldEntries[places[0].indexInTopLevel]
}

func (m *snapshot) arrangementOfGroup(group int32) int32 {
	return m.topLevels[m.placesOf(group)[0].topLevelIndex].arrangement
}

func (m *snapshot) parentGroup(group int32) int32 {
	if g, ok := m.groupNowUnder[group]; ok {
		return g
	}
	p := m.placesOf(group)[0]
	return m.topLevels[p.topLevelIndex].parentOf(p.indexInTopLevel)
}

func (m *snapshot) fetchedDirectlyUnder(entry, group int32) bool {
	for _, p := range m.placesOf(entry) {
		if m.topLevels[p.topLevelIndex].parentOf(p.indexInTopLevel) == group {
			return true
		}
	}
	return false
}

func (m *snapshot) findPlaces() {
	m.placesStart = make([]int32, m.count()+1)
	for _, t := range m.topLevels {
		if t != nil {
			for _, n := range t.entries {
				m.placesStart[n+1]++
			}
		}
	}
	for i := 1; i < len(m.placesStart); i++ {
		m.placesStart[i] += m.placesStart[i-1]
	}
	m.places = make([]position, m.placesStart[m.count()])
	nextFree := slices.Clone(m.placesStart[:m.count()])
	for index, t := range m.topLevels {
		if t != nil {
			for i, n := range t.entries {
				m.places[nextFree[n]] = position{int32(index), int32(i)}
				nextFree[n]++
			}
		}
	}
}

func (m *snapshot) buildLinks(edits map[pair]link) {
	type providersSay struct {
		kind     LinkKind
		disagree bool
		counted  bool
	}
	say := map[pair]*providersSay{}
	total := len(edits)
	for _, mapping := range m.mappings {
		if mapping == nil {
			continue
		}
		total += len(mapping.links)
		for _, k := range mapping.links {
			if s, ok := say[k.pair]; !ok {
				say[k.pair] = &providersSay{kind: k.kind}
			} else if s.kind != k.kind {
				s.disagree = true
			}
		}
	}

	m.linksStart = make([]int32, m.count()+1)
	count := func(p pair) {
		m.linksStart[p.lower+1]++
		m.linksStart[p.higher+1]++
	}
	for p := range edits {
		count(p)
	}
	for _, mapping := range m.mappings {
		if mapping != nil {
			for _, k := range mapping.links {
				count(k.pair)
			}
		}
	}
	for i := 1; i < len(m.linksStart); i++ {
		m.linksStart[i] += m.linksStart[i-1]
	}
	m.links = make([]linkEnd, m.linksStart[m.count()])
	m.linkWhy = make([][]Evidence, 0, total)
	nextFree := slices.Clone(m.linksStart[:m.count()])
	add := func(p pair, kind LinkKind, why []Evidence, mapping int32, counts bool) {
		i := int32(len(m.linkWhy))
		m.linkWhy = append(m.linkWhy, why)
		m.links[nextFree[p.lower]] = linkEnd{p.higher, mapping, i, kind, counts}
		nextFree[p.lower]++
		m.links[nextFree[p.higher]] = linkEnd{p.lower, mapping, i, kind.seenFromOtherSide(), counts}
		nextFree[p.higher]++
	}
	for p, k := range edits {
		add(p, k.kindSeenFromLower, k.why, none, true)
	}
	for index, mapping := range m.mappings {
		if mapping == nil {
			continue
		}
		for _, k := range mapping.links {
			s := say[k.pair]
			_, edited := edits[k.pair]
			counts := !edited && !s.disagree && !s.counted
			s.counted = s.counted || counts
			add(k.pair, k.kind, k.why, int32(index), counts)
		}
	}
	for p, s := range say {
		if _, edited := edits[p]; s.disagree && !edited {
			m.disagreements = append(m.disagreements, p)
		}
	}
	slices.SortFunc(m.disagreements, func(x, y pair) int { return cmp.Or(cmp.Compare(x.lower, y.lower), cmp.Compare(x.higher, y.higher)) })
	for n := range m.count() {
		if ends := m.links[m.linksStart[n]:m.linksStart[n+1]]; len(ends) > 1 {
			slices.SortFunc(ends, func(x, y linkEnd) int {
				return cmp.Or(cmp.Compare(x.other, y.other), cmp.Compare(x.mapping, y.mapping))
			})
		}
	}
}

func (m *snapshot) linksOf(n int32) []linkEnd {
	if int(n) >= m.count() {
		return nil
	}
	return m.links[m.linksStart[n]:m.linksStart[n+1]]
}

func (m *snapshot) buildItems() {
	count := m.count()
	sets := newDisjointSets(count)
	isItemEntry := make([]bool, count)
	for n := range int32(count) {
		isItemEntry[n] = m.isFetched(n) && !m.isGroup(n)
	}
	for n := range int32(count) {
		if !isItemEntry[n] {
			continue
		}
		for _, k := range m.linksOf(n) {
			if k.countsForItems && k.kindSeenFromThisEntry == Same && k.other > n && isItemEntry[k.other] {
				sets.join(n, k.other)
			}
		}
	}
	m.itemOf = make([]int32, count)
	m.membersStart = make([]int32, count+1)
	for n := range int32(count) {
		m.itemOf[n] = none
		if isItemEntry[n] {
			m.itemOf[n] = sets.find(n)
			m.membersStart[m.itemOf[n]+1]++
		}
	}
	for i := 1; i < len(m.membersStart); i++ {
		m.membersStart[i] += m.membersStart[i-1]
	}
	m.members = make([]int32, m.membersStart[count])
	nextFree := slices.Clone(m.membersStart[:count])
	for n := range int32(count) {
		if item := m.itemOf[n]; item != none {
			m.members[nextFree[item]] = n
			nextFree[item]++
		}
	}
}

func (m *snapshot) membersOf(item int32) []int32 {
	return m.members[m.membersStart[item]:m.membersStart[item+1]]
}

func (m *snapshot) applyAdds(adds map[int32][]addToGroup) {
	m.entriesAddedUnder = map[int32][]int32{}
	m.addedAway = map[entryInArrangement]struct{}{}
	m.groupNowUnder = map[int32]int32{}
	type add struct {
		entry int32
		addToGroup
	}
	var newestFirst []add
	for entry, list := range adds {
		for _, a := range list {
			newestFirst = append(newestFirst, add{entry, a})
		}
	}
	slices.SortFunc(newestFirst, func(x, y add) int { return cmp.Compare(y.editOrder, x.editOrder) })

	var toApply []add
	for _, a := range newestFirst {
		entry, group := a.entry, a.group
		if !m.isFetched(entry) || !m.isGroup(group) {
			m.unappliedEdits = append(m.unappliedEdits, entry)
			continue
		}
		key := entryInArrangement{m.arrangementOfGroup(group), entry}
		if _, older := m.addedAway[key]; older {
			continue
		}
		m.addedAway[key] = struct{}{}
		toApply = append(toApply, a)
	}
	slices.Reverse(toApply)
	for _, a := range toApply {
		entry, group := a.entry, a.group
		arrangement := m.arrangementOfGroup(group)
		if m.isGroup(entry) {
			underItself := false
			for x := group; x != none && !underItself; x = m.parentGroup(x) {
				underItself = x == entry
			}
			if underItself || m.arrangementOfGroup(entry) != arrangement {
				delete(m.addedAway, entryInArrangement{arrangement, entry})
				m.unappliedEdits = append(m.unappliedEdits, entry)
				continue
			}
			m.groupNowUnder[entry] = group
		}
		m.entriesAddedUnder[group] = append(m.entriesAddedUnder[group], entry)
	}
}

func (m *snapshot) entriesDirectlyUnder(top *topLevel, index int32) iter.Seq[int32] {
	return func(yield func(int32) bool) {
		end := index + top.allEntriesUnderCount[index] + 1
		for i := index + 1; i < end; i += top.allEntriesUnderCount[i] + 1 {
			n := top.entries[i]
			if _, away := m.addedAway[entryInArrangement{top.arrangement, n}]; !away && !yield(n) {
				return
			}
		}
		for _, n := range m.entriesAddedUnder[top.entries[index]] {
			if !yield(n) {
				return
			}
		}
	}
}

func (m *snapshot) findUndecidedAndConflicts() {
	for n := range int32(m.count()) {
		wholeUndecided := false
		for _, k := range m.linksOf(n) {
			if !k.countsForItems {
				continue
			}
			switch {
			case k.kindSeenFromThisEntry == NotSame && k.other > n && m.itemOf[n] != none && m.itemOf[n] == m.itemOf[k.other]:
				m.conflicts = append(m.conflicts, pair{n, k.other})
			case k.kindSeenFromThisEntry == Contains && m.isFetched(n) && m.isFetched(k.other):
				wholeChosen, decided := m.choices[n]
				if !decided {
					wholeUndecided = true
				} else if wholeChosen && m.choices[k.other] {
					m.conflicts = append(m.conflicts, pair{n, k.other})
				}
			}
		}
		if wholeUndecided {
			m.undecided = append(m.undecided, n)
		}
	}
}

func (m *snapshot) decideWorks() {
	competing := newDisjointSets(len(m.topLevels))
	for item := range int32(m.count()) {
		if m.itemOf[item] != item {
			continue
		}
		first := none
		for _, n := range m.membersOf(item) {
			for _, p := range m.placesOf(n) {
				if first == none {
					first = p.topLevelIndex
				} else {
					competing.join(first, p.topLevelIndex)
				}
			}
		}
	}

	byCompetition := map[int32][]int32{}
	var competitions []int32
	for index, t := range m.topLevels {
		if t == nil {
			continue
		}
		c := competing.find(int32(index))
		if _, seen := byCompetition[c]; !seen {
			competitions = append(competitions, c)
		}
		byCompetition[c] = append(byCompetition[c], int32(index))
	}
	undecidedCompetition := make([]bool, len(m.topLevels))
	for _, c := range competitions {
		var chosen, candidates []int32
		for _, index := range byCompetition[c] {
			c, decided := m.choices[m.topLevels[index].entries[0]]
			switch {
			case decided && c:
				chosen = append(chosen, index)
			case !decided:
				candidates = append(candidates, index)
			}
		}
		switch {
		case len(chosen) > 0:
			m.works = append(m.works, chosen...)
			for _, other := range chosen[1:] {
				m.conflicts = append(m.conflicts, pair{m.topLevels[chosen[0]].entries[0], m.topLevels[other].entries[0]})
			}
		case len(candidates) == 1:
			m.works = append(m.works, candidates[0])
		case len(candidates) > 1:
			undecidedCompetition[c] = true
			for _, index := range candidates {
				m.undecided = append(m.undecided, m.topLevels[index].entries[0])
			}
		}
	}
	slices.Sort(m.works)

	placed := make([]bool, m.count())
	var place func(group int32)
	place = func(group int32) {
		p := m.placesOf(group)[0]
		for n := range m.entriesDirectlyUnder(m.topLevels[p.topLevelIndex], p.indexInTopLevel) {
			if !placed[n] {
				placed[n] = true
				if m.isGroup(n) {
					place(n)
				}
			}
		}
	}
	for _, index := range m.works {
		top := m.topLevels[index].entries[0]
		placed[top] = true
		if m.isGroup(top) {
			place(top)
		}
	}
	for item := range int32(m.count()) {
		if m.itemOf[item] != item {
			continue
		}
		members := m.membersOf(item)
		if slices.ContainsFunc(members, func(n int32) bool { return placed[n] }) {
			continue
		}
		if undecidedCompetition[competing.find(m.placesOf(members[0])[0].topLevelIndex)] {
			continue
		}
		m.unplaced = append(m.unplaced, item)
	}
}

func (m *snapshot) listUnappliedEdits() {
	for key := range m.valueEdits {
		if !m.isFetched(key.entry) {
			m.unappliedEdits = append(m.unappliedEdits, key.entry)
		}
	}
	for n := range m.choices {
		if !m.isFetched(n) {
			m.unappliedEdits = append(m.unappliedEdits, n)
		}
	}
	for n := range int32(m.count()) {
		if !m.isFetched(n) && slices.ContainsFunc(m.linksOf(n), func(k linkEnd) bool { return k.mapping == none }) {
			m.unappliedEdits = append(m.unappliedEdits, n)
		}
	}
	slices.Sort(m.unappliedEdits)
	m.unappliedEdits = slices.Compact(m.unappliedEdits)
}

type disjointSets []int32

func newDisjointSets(count int) disjointSets {
	s := make(disjointSets, count)
	for i := range s {
		s[i] = int32(i)
	}
	return s
}

func (s disjointSets) find(n int32) int32 {
	for s[n] != n {
		s[n] = s[s[n]]
		n = s[n]
	}
	return n
}

func (s disjointSets) join(a, b int32) {
	a, b = s.find(a), s.find(b)
	if a != b {
		s[max(a, b)] = min(a, b)
	}
}
