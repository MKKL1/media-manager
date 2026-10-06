package library

import "slices"

func (l *Library) putValue(v Value) (check func() error, apply func()) {
	return func() error {
			if err := checkID(v.Entry); err != nil {
				return err
			}
			if v.Text == "" {
				return refused("an empty value for %v; Remove takes a value back", v.Entry)
			}
			if err := l.checkNewField(v.Field); err != nil {
				return err
			}
			n, known := l.entryNumbers.lookUp(v.Entry)
			if f, ok := l.fieldNumbers[v.Field]; ok && known && l.valueEdits[entryAndField{n, f}] == v.Text {
				return errNothingToDo
			}
			return nil
		},
		func() { l.valueEdits[entryAndField{l.numberEntry(v.Entry), l.fieldNumber(v.Field)}] = v.Text }
}

func (l *Library) removeValue(v Value) (check func() error, apply func()) {
	key := func() (entryAndField, bool) {
		n, ok := l.entryNumbers.lookUp(v.Entry)
		f, known := l.fieldNumbers[v.Field]
		return entryAndField{n, f}, ok && known
	}
	return func() error {
			if k, ok := key(); !ok || l.valueEdits[k] == "" {
				return errNothingToDo
			}
			return nil
		},
		func() {
			if k, ok := key(); ok {
				delete(l.valueEdits, k)
			}
		}
}

func (l *Library) putMove(mv Move) (check func() error, apply func()) {
	return func() error {
			return l.checkMove(mv)
		},
		func() {
			entry, group := l.numberEntry(mv.Entry), l.numberEntry(mv.Group)
			l.editsMade++
			moves := slices.DeleteFunc(l.addsToGroups[entry], func(a addToGroup) bool {
				return a.group == group || l.sameArrangement(a.group, group)
			})
			l.addsToGroups[entry] = append(moves, addToGroup{group, l.editsMade})
		}
}

func (l *Library) sameArrangement(group, other int32) bool {
	a, b := l.entryInfo[group].groupTopLevelIndex, l.entryInfo[other].groupTopLevelIndex
	return a != none && b != none && l.topLevels[a].arrangement == l.topLevels[b].arrangement
}

func (l *Library) checkMove(mv Move) error {
	if err := checkID(mv.Group); err != nil {
		return err
	}
	if err := checkID(mv.Entry); err != nil {
		return err
	}
	if mv.Group.Provider != mv.Entry.Provider {
		return refused("%v can't go under %v, another provider's group", mv.Entry, mv.Group)
	}
	if mv.Group == mv.Entry {
		return refused("%v can't go under itself", mv.Entry)
	}
	group, entry := l.numberEntry(mv.Group), l.numberEntry(mv.Entry)
	if slices.ContainsFunc(l.addsToGroups[entry], func(a addToGroup) bool { return a.group == group }) {
		return errNothingToDo
	}
	groupInfo, entryInfo := l.entryInfo[group], l.entryInfo[entry]
	if groupInfo.fetchedUnderCount == 0 {
		return nil
	}
	if !groupInfo.canHoldEntries {
		return refused("%v can't go under %v, which can't hold entries", mv.Entry, mv.Group)
	}
	if entryInfo.fetchedUnderCount == 0 || !entryInfo.canHoldEntries {
		return nil
	}
	m := l.snapshotWhileWriting()
	if m.arrangementOfGroup(entry) != m.arrangementOfGroup(group) {
		return refused("%v can't go under %v, a group of another arrangement", mv.Entry, mv.Group)
	}
	for x := group; x != none; x = m.parentGroup(x) {
		if x == entry {
			return refused("%v can't go under %v, which is under it", mv.Entry, mv.Group)
		}
	}
	return nil
}

func (l *Library) removeMove(mv Move) (check func() error, apply func()) {
	isMove := func(group int32) func(addToGroup) bool {
		return func(a addToGroup) bool { return a.group == group }
	}
	numbers := func() (entry, group int32, ok bool) {
		entry, okEntry := l.entryNumbers.lookUp(mv.Entry)
		group, okGroup := l.entryNumbers.lookUp(mv.Group)
		return entry, group, okEntry && okGroup
	}
	return func() error {
			entry, group, ok := numbers()
			if !ok || !slices.ContainsFunc(l.addsToGroups[entry], isMove(group)) {
				return errNothingToDo
			}
			return nil
		},
		func() {
			entry, group, ok := numbers()
			if !ok {
				return
			}
			moves := slices.DeleteFunc(l.addsToGroups[entry], isMove(group))
			if len(moves) == 0 {
				delete(l.addsToGroups, entry)
				return
			}
			l.addsToGroups[entry] = moves
		}
}

func checkLinkKind(k LinkKind) error {
	if k < Same || k > Contains {
		return refused("%v is not a kind of link", k)
	}
	return nil
}

func (l *Library) checkLinkEnds(k Link) error {
	for _, x := range [2]ID{k.From, k.To} {
		if err := checkID(x); err != nil {
			return err
		}
		if n, ok := l.entryNumbers.lookUp(x); ok && int(n) < len(l.entryInfo) {
			if info := l.entryInfo[n]; info.fetchedUnderCount > 0 && info.canHoldEntries {
				return refused("%v can hold entries; only entries that can't are linked", x)
			}
		}
	}
	if k.From == k.To {
		return refused("%v is linked to itself", k.From)
	}
	return nil
}

func (l *Library) pairOf(k Link) (pair, LinkKind) {
	a, b := l.numberEntry(k.From), l.numberEntry(k.To)
	if a > b {
		return pair{b, a}, k.Kind.seenFromOtherSide()
	}
	return pair{a, b}, k.Kind
}

func (l *Library) putLink(k Link) (check func() error, apply func()) {
	return func() error {
			if err := checkLinkKind(k.Kind); err != nil {
				return err
			}
			if err := l.checkLinkEnds(k); err != nil {
				return err
			}
			p, kind := l.pairOf(k)
			if current, ok := l.links[p]; ok && current.kindSeenFromLower == kind && slices.Equal(current.why, k.Why) {
				return errNothingToDo
			}
			return nil
		},
		func() {
			p, kind := l.pairOf(k)
			l.links[p] = link{kind, slices.Clone(k.Why)}
		}
}

func (l *Library) removeLink(k Link) (check func() error, apply func()) {
	key := func() (pair, bool) {
		a, okA := l.entryNumbers.lookUp(k.From)
		b, okB := l.entryNumbers.lookUp(k.To)
		return pair{min(a, b), max(a, b)}, okA && okB
	}
	return func() error {
			p, ok := key()
			if _, edited := l.links[p]; !ok || !edited {
				return errNothingToDo
			}
			return nil
		},
		func() {
			if p, ok := key(); ok {
				delete(l.links, p)
			}
		}
}

func (l *Library) putChoice(c Choice) (check func() error, apply func()) {
	return func() error {
			if err := checkID(c.Entry); err != nil {
				return err
			}
			if n, ok := l.entryNumbers.lookUp(c.Entry); ok {
				if current, decided := l.choices[n]; decided && current == c.Chosen {
					return errNothingToDo
				}
			}
			return nil
		},
		func() { l.choices[l.numberEntry(c.Entry)] = c.Chosen }
}

func (l *Library) removeChoice(c Choice) (check func() error, apply func()) {
	return func() error {
			n, ok := l.entryNumbers.lookUp(c.Entry)
			if _, decided := l.choices[n]; !ok || !decided {
				return errNothingToDo
			}
			return nil
		},
		func() {
			if n, ok := l.entryNumbers.lookUp(c.Entry); ok {
				delete(l.choices, n)
			}
		}
}
