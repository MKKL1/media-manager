package postgres_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"server/internal/event"
	"server/internal/library"
	"server/internal/library/postgres"
)

const (
	tmdb   library.Provider = "tmdb"
	tvmaze library.Provider = "tvmaze"
	title  library.Field    = "title"
	note   library.Field    = "note"
)

func entryID(p library.Provider, s string) library.ID { return library.ID{Provider: p, ProviderID: s} }

func episode(s, t string) library.Tree {
	return library.Tree{ProviderID: s, Values: map[library.Field]string{title: t}}
}

func holder(s string, entries ...library.Tree) library.Tree {
	return library.Tree{ProviderID: s, CanHoldEntries: true, Values: map[library.Field]string{title: s}, Entries: entries}
}

func fetched(p library.Provider, arrangement string, top library.Tree) library.Fetched {
	return library.Fetched{Provider: p, Arrangement: arrangement, Top: top.ProviderID, CanHoldEntries: top.CanHoldEntries, Values: top.Values, Entries: top.Entries}
}

func dump(lib *library.Library) string {
	var b strings.Builder
	var entry func(e library.Entry, depth int)
	entry = func(e library.Entry, depth int) {
		pad := strings.Repeat("  ", depth)
		fmt.Fprintf(&b, "%s%v", pad, e)
		for _, f := range []library.Field{title, note} {
			for s, text := range e.Values(f) {
				fmt.Fprintf(&b, " %v=%q(%T)", f, text, s)
			}
		}
		var statements []string
		for s := range e.Statements() {
			statements = append(statements, fmt.Sprintf("%T%+v", s, s))
		}
		slices.Sort(statements)
		fmt.Fprintf(&b, " statements=%v", statements)
		if it, ok := e.AsItem(); ok {
			var members []string
			for m := range it.Entries() {
				members = append(members, m.String())
			}
			slices.Sort(members)
			fmt.Fprintf(&b, " item=%v", members)
		}
		b.WriteString("\n")
		if g, ok := e.AsGroup(); ok {
			for c := range g.Entries() {
				entry(c, depth+1)
			}
		}
	}
	for _, p := range []library.Provider{tmdb, tvmaze} {
		for a := range lib.Arrangements(p) {
			fmt.Fprintf(&b, "== %v %v\n", p, a.Name())
			for top := range a.Top() {
				entry(top, 0)
			}
		}
	}
	for r := range lib.Reports() {
		fmt.Fprintf(&b, "report %v %v\n", r.Kind, r.Entries)
	}
	return b.String()
}

func openTestDB(t *testing.T) *library.Library {
	t.Helper()
	ctx := context.Background()
	db := openDB(t)
	store := postgres.New(db)
	lib, err := library.Open(ctx, store, &event.Bus{})
	require.NoError(t, err)
	reopen = func() *library.Library {
		l, err := library.Open(ctx, store, &event.Bus{})
		require.NoError(t, err)
		return l
	}
	return lib
}

var reopen func() *library.Library

func TestLibraryStoreReplaysToSameLibrary(t *testing.T) {
	ctx := context.Background()
	lib := openTestDB(t)
	put := func(statements ...library.Statement) {
		t.Helper()
		for _, s := range statements {
			require.NoError(t, lib.Put(ctx, s))
		}
	}
	remove := func(s library.Statement) {
		t.Helper()
		removed, err := lib.Remove(ctx, s)
		require.NoError(t, err)
		require.True(t, removed, "%+v", s)
	}

	show := func(ep1 string) library.Fetched {
		return fetched(tmdb, "aired", holder("tv:1",
			holder("s1", episode("e1", ep1), episode("e2", "Train Job")),
			holder("s2", episode("e3", "Ariel"))))
	}
	put(show("Serenity"), fetched(tmdb, "aired", holder("tv:9", episode("x1", "Gone Soon"))), show("Serenity (pilot)"))
	remove(library.Fetched{Provider: tmdb, Arrangement: "aired", Top: "tv:9"})

	put(fetched(tmdb, "dvd", holder("dvd:1", holder("d1", episode("e2", "Train Job")))))

	put(fetched(tvmaze, "aired", holder("m1", holder("ms1", episode("a", "Serenity"), episode("b", "Train Job")))))

	put(library.Value{Entry: entryID(tmdb, "e1"), Field: title, Text: "Pilot"}, library.Value{Entry: entryID(tmdb, "e1"), Field: note, Text: "tmp"})
	remove(library.Value{Entry: entryID(tmdb, "e1"), Field: note})

	move := func(entry, group string) library.Move {
		return library.Move{Entry: entryID(tmdb, entry), Group: entryID(tmdb, group)}
	}
	put(move("e3", "s1"), move("e1", "s2"), move("e3", "s2"), move("e3", "s1"))

	put(
		library.Link{From: entryID(tvmaze, "a"), To: entryID(tmdb, "e1"), Kind: library.Same, Why: []library.Evidence{"same title", "same year"}},
		library.Link{From: entryID(tvmaze, "b"), To: entryID(tmdb, "e2"), Kind: library.NotSame})
	remove(library.Link{From: entryID(tmdb, "e2"), To: entryID(tvmaze, "b")})
	put(
		library.Link{From: entryID(tvmaze, "b"), To: entryID(tmdb, "e3"), Kind: library.Contains, Why: []library.Evidence{"two parts"}},
		library.Link{From: entryID(tmdb, "e3"), To: entryID(tvmaze, "ms1x"), Kind: library.Contains})

	providerSame := func(ownID string) library.Mapping {
		return library.Mapping{Provider: "anime-lists", OwnID: ownID, Links: []library.Link{
			{From: entryID(tvmaze, "b"), To: entryID(tmdb, "e2"), Kind: library.Same, Why: []library.Evidence{"mapped"}}}}
	}
	put(providerSame("anidb:1"), providerSame("anidb:2"), providerSame("anidb:1"))
	remove(library.Mapping{Provider: "anime-lists", OwnID: "anidb:2"})

	put(
		library.Choice{Entry: entryID(tmdb, "tv:1"), Chosen: true},
		library.Choice{Entry: entryID(tvmaze, "m1"), Chosen: true},
		library.Choice{Entry: entryID(tvmaze, "m1"), Chosen: false},
		library.Choice{Entry: entryID(tmdb, "dvd:1"), Chosen: false})
	remove(library.Choice{Entry: entryID(tmdb, "dvd:1")})

	before := dump(lib)
	after := dump(reopen())
	require.Equal(t, before, after)

	require.Contains(t, after, `title="Pilot"(library.Value)`)
	require.Contains(t, after, "Serenity (pilot)")
	require.NotContains(t, after, "Gone Soon")
	require.NotContains(t, after, "tmp")
	require.Contains(t, after, "library.Choice{Entry:tmdb:tv:1 Chosen:true}")
	require.Contains(t, after, "library.Choice{Entry:tvmaze:m1 Chosen:false}")
	require.NotContains(t, after, "library.Choice{Entry:tmdb:dvd:1")
	require.Contains(t, after, "library.Link{From:tvmaze:b To:tmdb:e3 Kind:contains Why:[two parts]}")
	require.Contains(t, after, "library.Link{From:tmdb:e3 To:tvmaze:ms1x Kind:contains")
	require.NotContains(t, after, "not same")
	require.Contains(t, after, "library.Mapping{Provider:anime-lists OwnID:anidb:1 Links:[{From:tvmaze:b To:tmdb:e2 Kind:same Why:[mapped]}]}")
	require.NotContains(t, after, "anidb:2")
	s1Line := strings.Index(after, "tmdb:s1 ")
	s2Line := strings.Index(after, "tmdb:s2 ")
	require.Less(t, s1Line, s2Line)
	require.Less(t, strings.Index(after, "tmdb:e3"), s2Line)
}

func openDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	name := fmt.Sprintf("%s_test_%d", "library", rand.Int63())
	_, err = admin.Exec(ctx, "CREATE DATABASE "+name)
	require.NoError(t, err)
	t.Cleanup(func() {
		admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
		admin.Close()
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.ConnConfig.Database = name
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(db.Close)
	require.NoError(t, postgres.Migrate(ctx, db))
	require.NoError(t, postgres.Migrate(ctx, db))
	return db
}
