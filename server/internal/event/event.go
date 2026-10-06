package event

import (
	"context"
	"crypto/rand"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Event struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Subject string    `json:"subject"`
	By      string    `json:"by,omitempty"`
	Time    time.Time `json:"time"`
	Data    any       `json:"data,omitempty"`

	seq uint64
}

var ErrTooOld = errors.New("too old to resume")

type byKey struct{}

func WithBy(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, byKey{}, name)
}

const (
	keptToResumeFrom      = 1024
	maxBehindBeforeCutOff = 64
)

type Bus struct {
	mu    sync.Mutex
	start string
	seq   uint64
	ring  [keptToResumeFrom]Event
	subs  map[*subscriber]struct{}
}

type subscriber struct {
	match func(Event) bool
	ch    chan Event
}

func (b *Bus) Publish(ctx context.Context, e Event) {
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	if e.By == "" {
		e.By, _ = ctx.Value(byKey{}).(string)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.init()
	b.seq++
	e.seq = b.seq
	e.ID = b.start + "-" + strconv.FormatUint(b.seq, 10)
	b.ring[b.seq%keptToResumeFrom] = e
	for s := range b.subs {
		if !s.match(e) {
			continue
		}
		select {
		case s.ch <- e:
		default:
			b.cutOff(s)
		}
	}
}

func (b *Bus) Subscribe(after string, match func(Event) bool) (events <-chan Event, cancel func(), err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.init()
	var missed []Event
	if after != "" {
		from, err := b.seqAfter(after)
		if err != nil {
			return nil, nil, err
		}
		for seq := from + 1; seq <= b.seq; seq++ {
			if e := b.ring[seq%keptToResumeFrom]; match(e) {
				missed = append(missed, e)
			}
		}
	}
	s := &subscriber{match: match, ch: make(chan Event, maxBehindBeforeCutOff+len(missed))}
	for _, e := range missed {
		s.ch <- e
	}
	b.subs[s] = struct{}{}
	return s.ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.cutOff(s)
	}, nil
}

func (b *Bus) init() {
	if b.subs == nil {
		b.start = rand.Text()[:8]
		b.subs = map[*subscriber]struct{}{}
	}
}

func (b *Bus) cutOff(s *subscriber) {
	if _, ok := b.subs[s]; ok {
		delete(b.subs, s)
		close(s.ch)
	}
}

func (b *Bus) seqAfter(id string) (uint64, error) {
	start, n, ok := strings.Cut(id, "-")
	seq, err := strconv.ParseUint(n, 10, 64)
	oldest := uint64(1)
	if b.seq > keptToResumeFrom {
		oldest = b.seq - keptToResumeFrom + 1
	}
	if !ok || err != nil || start != b.start || seq > b.seq || seq+1 < oldest {
		return 0, ErrTooOld
	}
	return seq, nil
}
