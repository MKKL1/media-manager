package library

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"server/internal/event"
)

type Provider string

type Field string

type Evidence string

type ID struct {
	Provider   Provider
	ProviderID string
}

func (id ID) String() string { return string(id.Provider) + ":" + id.ProviderID }

func (id ID) MarshalText() ([]byte, error) { return []byte(id.String()), nil }

func (id *ID) UnmarshalText(b []byte) error {
	p, rest, ok := strings.Cut(string(b), ":")
	if !ok || p == "" || rest == "" {
		return fmt.Errorf("entry id %q is not <provider>:<id>", b)
	}
	*id = ID{Provider(p), rest}
	return nil
}

type Statement interface{ statement() }

type Fetched struct {
	Provider       Provider
	Arrangement    string
	Top            string
	CanHoldEntries bool
	Values         map[Field]string
	Entries        []Tree
}

type Tree struct {
	ProviderID     string
	CanHoldEntries bool
	Values         map[Field]string
	Entries        []Tree
}

type Mapping struct {
	Provider Provider
	OwnID    string
	Links    []Link
}

type Value struct {
	Entry ID
	Field Field
	Text  string
}

type Move struct {
	Entry, Group ID
}

type Link struct {
	From, To ID
	Kind     LinkKind
	Why      []Evidence
}

type Choice struct {
	Entry  ID
	Chosen bool
}

func (Fetched) statement() {}
func (Mapping) statement() {}
func (Value) statement()   {}
func (Move) statement()    {}
func (Link) statement()    {}
func (Choice) statement()  {}

type LinkKind uint8

const (
	Same LinkKind = iota + 1
	NotSame
	Contains
	partOf
)

func (k LinkKind) String() string {
	switch k {
	case Same:
		return "same"
	case NotSame:
		return "not same"
	case Contains:
		return "contains"
	case partOf:
		return "part of"
	}
	return fmt.Sprintf("LinkKind(%d)", k)
}

func (k LinkKind) seenFromOtherSide() LinkKind {
	switch k {
	case partOf:
		return Contains
	case Contains:
		return partOf
	}
	return k
}

func linkSeenFrom(this, other ID, kind LinkKind, why []Evidence) Link {
	if kind == partOf {
		return Link{From: other, To: this, Kind: Contains, Why: why}
	}
	return Link{From: this, To: other, Kind: kind, Why: why}
}

var ErrRefused = errors.New("refused")

const none int32 = -1

type Store interface {
	Put(ctx context.Context, s Statement) error
	Remove(ctx context.Context, s Statement) error
	All(ctx context.Context) iter.Seq2[Statement, error]
}

type Library struct {
	store  Store
	events *event.Bus

	latestSnapshot   atomic.Pointer[snapshot]
	needsNewSnapshot atomic.Bool
	loadingFromStore atomic.Bool
	entryNumbers     entryNumbers

	writeLock sync.Mutex

	fieldNumbers       map[Field]uint16
	arrangementNames   []arrangementName
	arrangementNumbers map[arrangementName]int32
	topLevels          []*topLevel
	topLevelOfEntry    map[entryInArrangement]int32
	entryInfo          []entryInfo
	mappings           []*mapping
	mappingIndex       map[mappingName]int32
	valueEdits         map[entryAndField]string
	addsToGroups       map[int32][]addToGroup
	links              map[pair]link
	choices            map[int32]bool
	editsMade          int64
}

type arrangementName struct {
	provider Provider
	name     string
}

type entryInfo struct {
	fetchedUnderCount  int32
	groupTopLevelIndex int32
	canHoldEntries     bool
}

type entryInArrangement struct{ arrangement, entry int32 }

type entryAndField struct {
	entry int32
	field uint16
}

type addToGroup struct {
	group     int32
	editOrder int64
}

type pair struct{ lower, higher int32 }

type link struct {
	kindSeenFromLower LinkKind
	why               []Evidence
}

var errNothingToDo = errors.New("nothing to do")

func refused(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrRefused}, args...)...)
}

func Open(ctx context.Context, store Store, events *event.Bus) (*Library, error) {
	l := &Library{
		store:              store,
		events:             events,
		entryNumbers:       entryNumbers{byID: map[ID]int32{}},
		fieldNumbers:       map[Field]uint16{},
		arrangementNumbers: map[arrangementName]int32{},
		topLevelOfEntry:    map[entryInArrangement]int32{},
		mappingIndex:       map[mappingName]int32{},
		valueEdits:         map[entryAndField]string{},
		addsToGroups:       map[int32][]addToGroup{},
		links:              map[pair]link{},
		choices:            map[int32]bool{},
	}
	l.latestSnapshot.Store(l.newSnapshot())
	l.loadingFromStore.Store(true)
	for s, err := range store.All(ctx) {
		if err == nil {
			err = l.Put(ctx, s)
		}
		if err != nil {
			return nil, fmt.Errorf("load library: %w", err)
		}
	}
	l.writeLock.Lock()
	l.loadingFromStore.Store(false)
	l.latestSnapshot.Store(l.newSnapshot())
	l.needsNewSnapshot.Store(false)
	l.writeLock.Unlock()
	return l, nil
}

func (l *Library) Put(ctx context.Context, s Statement) error {
	var check func() error
	var apply func()
	switch s := s.(type) {
	case Fetched:
		check, apply = l.putFetched(s)
	case Mapping:
		check, apply = l.putMapping(s)
	case Value:
		check, apply = l.putValue(s)
	case Move:
		check, apply = l.putMove(s)
	case Link:
		check, apply = l.putLink(s)
	case Choice:
		check, apply = l.putChoice(s)
	default:
		return refused("%T is not a statement", s)
	}
	_, err := l.write(check, func() error { return l.saved(ctx, l.store.Put(ctx, s), s, false) }, apply)
	return err
}

func (l *Library) Remove(ctx context.Context, s Statement) (removed bool, err error) {
	var check func() error
	var apply func()
	switch s := s.(type) {
	case Fetched:
		check, apply = l.removeFetched(s)
	case Mapping:
		check, apply = l.removeMapping(s)
	case Value:
		check, apply = l.removeValue(s)
	case Move:
		check, apply = l.removeMove(s)
	case Link:
		check, apply = l.removeLink(s)
	case Choice:
		check, apply = l.removeChoice(s)
	default:
		return false, refused("%T is not a statement", s)
	}
	return l.write(check, func() error { return l.saved(ctx, l.store.Remove(ctx, s), s, true) }, apply)
}

func (l *Library) saved(ctx context.Context, err error, s Statement, deleted bool) error {
	if err == nil {
		l.publish(ctx, s, deleted)
	}
	return err
}

func (l *Library) write(check, save func() error, apply func()) (changed bool, err error) {
	l.writeLock.Lock()
	defer l.writeLock.Unlock()
	for len(l.entryInfo) < l.entryNumbers.count() {
		l.entryInfo = append(l.entryInfo, entryInfo{groupTopLevelIndex: none})
	}
	if !l.loadingFromStore.Load() {
		if err := check(); err != nil {
			if errors.Is(err, errNothingToDo) {
				return false, nil
			}
			return false, err
		}
		if err := save(); err != nil {
			return false, err
		}
	}
	apply()
	l.needsNewSnapshot.Store(true)
	return true, nil
}

func checkID(id ID) error {
	if id.Provider == "" || id.ProviderID == "" {
		return refused("%q is not an entry id", id.String())
	}
	return nil
}

func (l *Library) numberEntry(id ID) int32 {
	n := l.entryNumbers.numberFor(id)
	for int(n) >= len(l.entryInfo) {
		l.entryInfo = append(l.entryInfo, entryInfo{groupTopLevelIndex: none})
	}
	return n
}

func (l *Library) fieldNumber(f Field) uint16 {
	if n, ok := l.fieldNumbers[f]; ok {
		return n
	}
	next := make(map[Field]uint16, len(l.fieldNumbers)+1)
	for k, v := range l.fieldNumbers {
		next[k] = v
	}
	n := uint16(len(l.fieldNumbers))
	next[f] = n
	l.fieldNumbers = next
	return n
}

func (l *Library) checkNewField(f Field) error {
	if f == "" {
		return refused("a value has no field")
	}
	if _, ok := l.fieldNumbers[f]; !ok && len(l.fieldNumbers) > math.MaxUint16 {
		return refused("field %q is one more than the library holds", f)
	}
	return nil
}

type entryNumbers struct {
	lock sync.RWMutex
	byID map[ID]int32
	ids  []ID
}

func (numbers *entryNumbers) lookUp(id ID) (int32, bool) {
	numbers.lock.RLock()
	n, ok := numbers.byID[id]
	numbers.lock.RUnlock()
	return n, ok
}

func (numbers *entryNumbers) numberFor(id ID) int32 {
	if n, ok := numbers.lookUp(id); ok {
		return n
	}
	numbers.lock.Lock()
	defer numbers.lock.Unlock()
	if n, ok := numbers.byID[id]; ok {
		return n
	}
	n := int32(len(numbers.ids))
	numbers.byID[id] = n
	numbers.ids = append(numbers.ids, id)
	return n
}

func (numbers *entryNumbers) id(n int32) ID {
	numbers.lock.RLock()
	defer numbers.lock.RUnlock()
	return numbers.ids[n]
}

func (numbers *entryNumbers) count() int {
	numbers.lock.RLock()
	defer numbers.lock.RUnlock()
	return len(numbers.ids)
}

func (numbers *entryNumbers) all() []ID {
	numbers.lock.RLock()
	defer numbers.lock.RUnlock()
	return numbers.ids
}
