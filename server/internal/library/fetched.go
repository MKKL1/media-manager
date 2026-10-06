package library

import (
	"iter"
	"math"
)

type Arrangement struct {
	library  *Library
	provider Provider
	name     string
}

func (l *Library) Arrangements(p Provider) iter.Seq[Arrangement] {
	m := l.latest()
	return func(yield func(Arrangement) bool) {
		fetchedIn := make([]bool, len(m.arrangementNames))
		for _, t := range m.topLevels {
			if t != nil {
				fetchedIn[t.arrangement] = true
			}
		}
		for number, name := range m.arrangementNames {
			if fetchedIn[number] && name.provider == p && !yield(Arrangement{l, name.provider, name.name}) {
				return
			}
		}
	}
}

func (a Arrangement) Provider() Provider { return a.provider }
func (a Arrangement) Name() string       { return a.name }
func (a Arrangement) String() string     { return string(a.provider) + " " + a.name }

func (a Arrangement) Top() iter.Seq[Entry] {
	m := a.library.latest()
	return func(yield func(Entry) bool) {
		for _, t := range m.topLevels {
			if t != nil && m.arrangementNames[t.arrangement] == (arrangementName{a.provider, a.name}) {
				if !yield(m.entry(t.entries[0])) {
					return
				}
			}
		}
	}
}

func (l *Library) putFetched(f Fetched) (check func() error, apply func()) {
	key := arrangementName{f.Provider, f.Arrangement}
	top := Tree{ProviderID: f.Top, CanHoldEntries: f.CanHoldEntries, Values: f.Values, Entries: f.Entries}
	return func() error { return l.checkPut(key, top) },
		func() {
			number, known := l.arrangementNumbers[key]
			if !known {
				number = int32(len(l.arrangementNames))
				l.arrangementNumbers[key] = number
				l.arrangementNames = append(l.arrangementNames, key)
			}
			replaced := l.topLevelIndexOf(key, ID{f.Provider, f.Top})
			t := l.newTopLevel(number, f.Provider, top)
			if replaced != none {
				l.removeFromEntryInfo(replaced)
				l.topLevels[replaced] = t
				l.addToEntryInfo(replaced)
				return
			}
			l.topLevels = append(l.topLevels, t)
			l.addToEntryInfo(int32(len(l.topLevels) - 1))
		}
}

func (l *Library) removeFetched(f Fetched) (check func() error, apply func()) {
	key, top := arrangementName{f.Provider, f.Arrangement}, ID{f.Provider, f.Top}
	return func() error {
			if l.topLevelIndexOf(key, top) == none {
				return errNothingToDo
			}
			return nil
		},
		func() {
			index := l.topLevelIndexOf(key, top)
			if index == none {
				return
			}
			l.removeFromEntryInfo(index)
			l.topLevels[index] = nil
		}
}

func (l *Library) topLevelIndexOf(key arrangementName, top ID) int32 {
	number, ok := l.arrangementNumbers[key]
	if !ok {
		return none
	}
	n, ok := l.entryNumbers.lookUp(top)
	if !ok {
		return none
	}
	if index, ok := l.topLevelOfEntry[entryInArrangement{number, n}]; ok && l.topLevels[index].entries[0] == n {
		return index
	}
	return none
}

func (l *Library) checkPut(key arrangementName, top Tree) error {
	if key.provider == "" || key.name == "" {
		return refused("a fetch names no provider or arrangement")
	}
	topID := ID{key.provider, top.ProviderID}
	replaced := l.topLevelIndexOf(key, topID)
	number, known := l.arrangementNumbers[key]
	seen := map[string]bool{}
	newFields := map[Field]bool{}
	var textBytes int
	var check func(e Tree) error
	check = func(e Tree) error {
		id := ID{key.provider, e.ProviderID}
		switch {
		case e.ProviderID == "":
			return refused("an entry under %v has no provider id", topID)
		case seen[e.ProviderID]:
			return refused("%v is fetched twice under %v", id, topID)
		case !e.CanHoldEntries && len(e.Entries) > 0:
			return refused("%v can't hold entries but is fetched with entries", id)
		}
		seen[e.ProviderID] = true
		for f, v := range e.Values {
			textBytes += len(v)
			if _, ok := l.fieldNumbers[f]; !ok {
				newFields[f] = true
			}
		}
		if n, ok := l.entryNumbers.lookUp(id); ok && int(n) < len(l.entryInfo) {
			info := l.entryInfo[n]
			index, inArrangement := l.topLevelOfEntry[entryInArrangement{number, n}]
			if known && inArrangement && index != replaced {
				return refused("%v is already under another top-level entry of %v %v", id, key.provider, key.name)
			}
			if e.CanHoldEntries && info.groupTopLevelIndex != none && info.groupTopLevelIndex != replaced {
				return refused("%v is a group already in another arrangement", id)
			}
			inReplaced := int32(0)
			if known && inArrangement && index == replaced {
				inReplaced = 1
			}
			if info.fetchedUnderCount > inReplaced && info.canHoldEntries != e.CanHoldEntries {
				return refused("%v can hold entries in one arrangement and not in another", id)
			}
		}
		for _, c := range e.Entries {
			if err := check(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := check(top); err != nil {
		return err
	}
	if len(l.fieldNumbers)+len(newFields) > math.MaxUint16+1 {
		return refused("%v brings more fields than the library holds (%d)", topID, math.MaxUint16+1)
	}
	if textBytes > math.MaxUint32 {
		return refused("%v has more than 4 GB of value text", topID)
	}
	return nil
}

func (l *Library) addToEntryInfo(index int32) {
	t := l.topLevels[index]
	for i, n := range t.entries {
		info := &l.entryInfo[n]
		info.fetchedUnderCount++
		info.canHoldEntries = t.canHoldEntries[i]
		if t.canHoldEntries[i] {
			info.groupTopLevelIndex = index
		}
		l.topLevelOfEntry[entryInArrangement{t.arrangement, n}] = index
	}
}

func (l *Library) removeFromEntryInfo(index int32) {
	t := l.topLevels[index]
	for _, n := range t.entries {
		info := &l.entryInfo[n]
		info.fetchedUnderCount--
		if info.groupTopLevelIndex == index {
			info.groupTopLevelIndex = none
		}
		if key := (entryInArrangement{t.arrangement, n}); l.topLevelOfEntry[key] == index {
			delete(l.topLevelOfEntry, key)
		}
	}
}
