package postgres

//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate

import (
	"cmp"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"iter"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"server/internal/library"
	"server/internal/library/postgres/queries"
)

//go:embed schema.sql
var schema string

func Migrate(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, schema); err != nil {
		return fmt.Errorf("migrate library: %w", err)
	}
	return nil
}

type Store struct {
	db *pgxpool.Pool
	q  *queries.Queries
}

var _ library.Store = (*Store)(nil)

func New(db *pgxpool.Pool) *Store { return &Store{db, queries.New(db)} }

type fetchedTree struct {
	ProviderID     string            `json:"id"`
	CanHoldEntries bool              `json:"can_hold_entries,omitempty"`
	Values         map[string]string `json:"values,omitempty"`
	Entries        []fetchedTree     `json:"entries,omitempty"`
}

func treeOf(t library.Tree) fetchedTree {
	saved := fetchedTree{ProviderID: t.ProviderID, CanHoldEntries: t.CanHoldEntries}
	if len(t.Values) > 0 {
		saved.Values = make(map[string]string, len(t.Values))
		for field, v := range t.Values {
			saved.Values[string(field)] = v
		}
	}
	for _, e := range t.Entries {
		saved.Entries = append(saved.Entries, treeOf(e))
	}
	return saved
}

func (t fetchedTree) tree() library.Tree {
	tree := library.Tree{ProviderID: t.ProviderID, CanHoldEntries: t.CanHoldEntries}
	if len(t.Values) > 0 {
		tree.Values = make(map[library.Field]string, len(t.Values))
		for field, v := range t.Values {
			tree.Values[library.Field(field)] = v
		}
	}
	for _, e := range t.Entries {
		tree.Entries = append(tree.Entries, e.tree())
	}
	return tree
}

var linkKinds = map[string]library.LinkKind{
	library.Same.String():     library.Same,
	library.NotSame.String():  library.NotSame,
	library.Contains.String(): library.Contains,
}

func linkInOrder(k library.Link) (a, b library.ID, kindFromA string) {
	if k.From.Provider < k.To.Provider || (k.From.Provider == k.To.Provider && k.From.ProviderID < k.To.ProviderID) {
		return k.From, k.To, k.Kind.String()
	}
	if k.Kind == library.Contains {
		return k.To, k.From, "part of"
	}
	return k.To, k.From, k.Kind.String()
}

func linkOf(a, b library.ID, kind string, why []string) (library.Link, error) {
	k := library.Link{From: a, To: b}
	if kind == "part of" {
		k.From, k.To, kind = b, a, library.Contains.String()
	}
	var ok bool
	if k.Kind, ok = linkKinds[kind]; !ok {
		return library.Link{}, fmt.Errorf("unknown link kind %q", kind)
	}
	for _, w := range why {
		k.Why = append(k.Why, library.Evidence(w))
	}
	return k, nil
}

func evidence(why []library.Evidence) []string {
	out := make([]string, len(why))
	for i, w := range why {
		out[i] = string(w)
	}
	return out
}

type linkRow struct {
	EntryProvider string   `json:"entry_provider"`
	EntryID       string   `json:"entry_id"`
	OtherProvider string   `json:"other_provider"`
	OtherID       string   `json:"other_id"`
	Kind          string   `json:"kind"`
	Why           []string `json:"why,omitempty"`
}

func (s *Store) Put(ctx context.Context, st library.Statement) error {
	var err error
	switch st := st.(type) {
	case library.Fetched:
		var content []byte
		top := library.Tree{ProviderID: st.Top, CanHoldEntries: st.CanHoldEntries, Values: st.Values, Entries: st.Entries}
		if content, err = json.Marshal(treeOf(top)); err == nil {
			err = s.q.PutFetched(ctx, queries.PutFetchedParams{Provider: string(st.Provider), Arrangement: st.Arrangement, TopID: st.Top, Content: content})
		}
	case library.Mapping:
		rows := make([]linkRow, len(st.Links))
		for i, k := range st.Links {
			a, b, kind := linkInOrder(k)
			rows[i] = linkRow{string(a.Provider), a.ProviderID, string(b.Provider), b.ProviderID, kind, evidence(k.Why)}
		}
		var content []byte
		if content, err = json.Marshal(rows); err == nil {
			err = s.q.PutMapping(ctx, queries.PutMappingParams{Provider: string(st.Provider), OwnID: st.OwnID, Content: content})
		}
	case library.Value:
		err = s.q.SetValueEdit(ctx, queries.SetValueEditParams{Provider: string(st.Entry.Provider), EntryID: st.Entry.ProviderID, Field: string(st.Field), Value: st.Text})
	case library.Move:

		err = s.q.AddToGroup(ctx, queries.AddToGroupParams{Provider: string(st.Group.Provider), GroupID: st.Group.ProviderID, EntryID: st.Entry.ProviderID})
	case library.Link:
		a, b, kind := linkInOrder(st)
		err = s.q.PutLink(ctx, queries.PutLinkParams{
			AProvider: string(a.Provider), AID: a.ProviderID, BProvider: string(b.Provider), BID: b.ProviderID,
			Kind: kind, Why: evidence(st.Why),
		})
	case library.Choice:
		err = s.q.SetChoice(ctx, queries.SetChoiceParams{Provider: string(st.Entry.Provider), EntryID: st.Entry.ProviderID, Chosen: st.Chosen})
	default:
		err = fmt.Errorf("%T is not a statement", st)
	}
	if err != nil {
		return fmt.Errorf("put %+v: %w", st, err)
	}
	return nil
}

func (s *Store) Remove(ctx context.Context, st library.Statement) error {
	var err error
	switch st := st.(type) {
	case library.Fetched:
		err = s.q.RemoveFetched(ctx, queries.RemoveFetchedParams{Provider: string(st.Provider), Arrangement: st.Arrangement, TopID: st.Top})
	case library.Mapping:
		err = s.q.RemoveMapping(ctx, queries.RemoveMappingParams{Provider: string(st.Provider), OwnID: st.OwnID})
	case library.Value:
		err = s.q.ClearValueEdit(ctx, queries.ClearValueEditParams{Provider: string(st.Entry.Provider), EntryID: st.Entry.ProviderID, Field: string(st.Field)})
	case library.Move:
		err = s.q.RemoveFromGroup(ctx, queries.RemoveFromGroupParams{Provider: string(st.Group.Provider), GroupID: st.Group.ProviderID, EntryID: st.Entry.ProviderID})
	case library.Link:
		a, b, _ := linkInOrder(st)
		err = s.q.RemoveLink(ctx, queries.RemoveLinkParams{AProvider: string(a.Provider), AID: a.ProviderID, BProvider: string(b.Provider), BID: b.ProviderID})
	case library.Choice:
		err = s.q.ClearChoice(ctx, queries.ClearChoiceParams{Provider: string(st.Entry.Provider), EntryID: st.Entry.ProviderID})
	default:
		err = fmt.Errorf("%T is not a statement", st)
	}
	if err != nil {
		return fmt.Errorf("remove %+v: %w", st, err)
	}
	return nil
}

type saved struct {
	fetched  []queries.AllFetchedRow
	mappings []queries.AllMappingsRow
	values   []queries.LibraryValueEdit
	adds     []queries.LibraryAddsToGroup
	links    []queries.LibraryLink
	choices  []queries.LibraryChoice
}

func (s *Store) readAll(ctx context.Context) (saved, error) {
	var all saved
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return all, err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	if all.fetched, err = q.AllFetched(ctx); err != nil {
		return all, fmt.Errorf("fetched: %w", err)
	}
	if all.mappings, err = q.AllMappings(ctx); err != nil {
		return all, fmt.Errorf("mappings: %w", err)
	}
	if all.values, err = q.AllValueEdits(ctx); err != nil {
		return all, fmt.Errorf("value edits: %w", err)
	}
	if all.adds, err = q.AllAddsToGroups(ctx); err != nil {
		return all, fmt.Errorf("adds to groups: %w", err)
	}
	if all.links, err = q.AllLinks(ctx); err != nil {
		return all, fmt.Errorf("links: %w", err)
	}
	if all.choices, err = q.AllChoices(ctx); err != nil {
		return all, fmt.Errorf("choices: %w", err)
	}
	return all, nil
}

func (s *Store) All(ctx context.Context) iter.Seq2[library.Statement, error] {
	return func(yield func(library.Statement, error) bool) {
		statements, err := s.all(ctx)
		if err != nil {
			yield(nil, fmt.Errorf("load library: %w", err))
			return
		}
		for _, st := range statements {
			if !yield(st, nil) {
				return
			}
		}
	}
}

func (s *Store) all(ctx context.Context) ([]library.Statement, error) {
	all, err := s.readAll(ctx)
	if err != nil {
		return nil, err
	}
	var statements []library.Statement
	for _, row := range all.fetched {
		var saved fetchedTree
		if err := json.Unmarshal(row.Content, &saved); err != nil {
			return nil, fmt.Errorf("fetched %v %v: %w", row.Provider, row.Arrangement, err)
		}
		top := saved.tree()
		statements = append(statements, library.Fetched{
			Provider: library.Provider(row.Provider), Arrangement: row.Arrangement, Top: top.ProviderID,
			CanHoldEntries: top.CanHoldEntries, Values: top.Values, Entries: top.Entries,
		})
	}
	for _, row := range all.mappings {
		var rows []linkRow
		if err := json.Unmarshal(row.Content, &rows); err != nil {
			return nil, fmt.Errorf("mapping %s %s: %w", row.Provider, row.OwnID, err)
		}
		mapping := library.Mapping{Provider: library.Provider(row.Provider), OwnID: row.OwnID, Links: make([]library.Link, len(rows))}
		for i, r := range rows {
			a := library.ID{Provider: library.Provider(r.EntryProvider), ProviderID: r.EntryID}
			b := library.ID{Provider: library.Provider(r.OtherProvider), ProviderID: r.OtherID}
			if mapping.Links[i], err = linkOf(a, b, r.Kind, r.Why); err != nil {
				return nil, fmt.Errorf("mapping %s %s: %w", row.Provider, row.OwnID, err)
			}
		}
		statements = append(statements, mapping)
	}

	type savedEdit struct {
		saved int64
		edit  library.Statement
	}
	var edits []savedEdit
	for _, r := range all.values {
		id := library.ID{Provider: library.Provider(r.Provider), ProviderID: r.EntryID}
		edits = append(edits, savedEdit{r.Saved, library.Value{Entry: id, Field: library.Field(r.Field), Text: r.Value}})
	}
	for _, r := range all.adds {
		edits = append(edits, savedEdit{r.Saved, library.Move{
			Entry: library.ID{Provider: library.Provider(r.Provider), ProviderID: r.EntryID},
			Group: library.ID{Provider: library.Provider(r.Provider), ProviderID: r.GroupID},
		}})
	}
	for _, r := range all.links {
		a := library.ID{Provider: library.Provider(r.AProvider), ProviderID: r.AID}
		b := library.ID{Provider: library.Provider(r.BProvider), ProviderID: r.BID}
		k, err := linkOf(a, b, r.Kind, r.Why)
		if err != nil {
			return nil, fmt.Errorf("link %v %v: %w", a, b, err)
		}
		edits = append(edits, savedEdit{r.Saved, k})
	}
	for _, r := range all.choices {
		id := library.ID{Provider: library.Provider(r.Provider), ProviderID: r.EntryID}
		edits = append(edits, savedEdit{r.Saved, library.Choice{Entry: id, Chosen: r.Chosen}})
	}
	slices.SortFunc(edits, func(x, y savedEdit) int { return cmp.Compare(x.saved, y.saved) })
	for _, e := range edits {
		statements = append(statements, e.edit)
	}
	return statements, nil
}
