package library

import "strings"

type topLevel struct {
	arrangement          int32
	entries              []int32
	allEntriesUnderCount []int32
	canHoldEntries       []bool
	firstValue           []int32
	valueField           []uint16
	valueEnd             []uint32
	valueText            string
}

func (l *Library) newTopLevel(arrangement int32, provider Provider, top Tree) *topLevel {
	var entries, values, textBytes int
	var count func(e Tree)
	count = func(e Tree) {
		entries++
		for _, v := range e.Values {
			if v != "" {
				values++
				textBytes += len(v)
			}
		}
		for _, c := range e.Entries {
			count(c)
		}
	}
	count(top)

	t := &topLevel{
		arrangement:          arrangement,
		entries:              make([]int32, 0, entries),
		allEntriesUnderCount: make([]int32, 0, entries),
		canHoldEntries:       make([]bool, 0, entries),
		firstValue:           make([]int32, 0, entries+1),
		valueField:           make([]uint16, 0, values),
		valueEnd:             make([]uint32, 0, values),
	}
	var text strings.Builder
	text.Grow(textBytes)
	var add func(e Tree)
	add = func(e Tree) {
		i := len(t.entries)
		t.entries = append(t.entries, l.numberEntry(ID{provider, e.ProviderID}))
		t.allEntriesUnderCount = append(t.allEntriesUnderCount, 0)
		t.canHoldEntries = append(t.canHoldEntries, e.CanHoldEntries)
		t.firstValue = append(t.firstValue, int32(len(t.valueField)))
		for f, v := range e.Values {
			if v != "" {
				t.valueField = append(t.valueField, l.fieldNumber(f))
				text.WriteString(v)
				t.valueEnd = append(t.valueEnd, uint32(text.Len()))
			}
		}
		for _, c := range e.Entries {
			add(c)
		}
		t.allEntriesUnderCount[i] = int32(len(t.entries) - i - 1)
	}
	add(top)
	t.firstValue = append(t.firstValue, int32(len(t.valueField)))
	t.valueText = text.String()
	return t
}

func (t *topLevel) value(i int32, field uint16) (string, bool) {
	for j := t.firstValue[i]; j < t.firstValue[i+1]; j++ {
		if t.valueField[j] == field {
			start := uint32(0)
			if j > 0 {
				start = t.valueEnd[j-1]
			}
			return t.valueText[start:t.valueEnd[j]], true
		}
	}
	return "", false
}

func (t *topLevel) parentOf(i int32) int32 {
	for j := i - 1; j >= 0; j-- {
		if j+t.allEntriesUnderCount[j] >= i {
			return t.entries[j]
		}
	}
	return none
}
