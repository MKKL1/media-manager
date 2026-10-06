package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"server/internal/event"
	"server/internal/user"
	"server/internal/user/postgres"
)

func openUserDB(t *testing.T) *postgres.Store {
	t.Helper()
	return postgres.New(openDB(t))
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
	name := fmt.Sprintf("%s_test_%d", "user", rand.Int63())
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

func TestUsersThroughPostgres(t *testing.T) {
	store := openUserDB(t)
	bus := &event.Bus{}
	published, _, err := bus.Subscribe("", func(event.Event) bool { return true })
	require.NoError(t, err)
	users := user.NewService(store, bus)
	ctx := event.WithBy(context.Background(), "admin")

	roles, err := users.Roles(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 3)
	whoami := user.Attributes{User: user.User{Name: "x", Groups: []string{user.Authenticated}}, Verb: "list", Resource: "whoami"}
	d, err := users.Authorize(ctx, whoami)
	require.NoError(t, err)
	require.Equal(t, user.Allow, d)

	require.NoError(t, users.PutUser(ctx, user.User{Name: "alice@example.com", Groups: []string{"family"}}))
	require.NoError(t, users.PutBinding(ctx, user.Binding{Name: "family-writes", Role: "writer", Subjects: []user.Subject{{Kind: "group", Name: "family"}}}))
	require.ErrorIs(t, users.PutUser(ctx, user.User{Name: "bad name"}), user.ErrInvalid)
	require.ErrorIs(t, users.PutRole(ctx, user.Role{Name: "r", Rules: []user.Rule{{Verbs: []string{"eat"}, Resources: []string{"*"}}}}), user.ErrInvalid)

	tok, secret, err := users.CreateToken(ctx, "alice@example.com", time.Time{})
	require.NoError(t, err)
	_, _, err = users.CreateToken(ctx, "nobody", time.Time{})
	require.ErrorIs(t, err, user.ErrNotFound)

	alice, ok, err := users.Authenticate(ctx, secret)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"family"}, alice.Groups)
	edit := user.Attributes{User: alice, Verb: "create", Resource: "entries/edits", Name: "tmdb:tv:1"}
	d, err = users.Authorize(ctx, edit)
	require.NoError(t, err)
	require.Equal(t, user.Allow, d)

	old, secretOld, err := users.CreateToken(ctx, "alice@example.com", time.Now().Add(-time.Minute))
	require.NoError(t, err)
	_, ok, _ = users.Authenticate(ctx, secretOld)
	require.False(t, ok, "expired token")

	tokens, err := users.Tokens(ctx)
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	require.Equal(t, tok, tokens[0])

	require.NoError(t, users.DeleteUser(ctx, "alice@example.com"))
	_, ok, _ = users.Authenticate(ctx, secret)
	require.False(t, ok)
	tokens, _ = users.Tokens(ctx)
	require.Empty(t, tokens)
	require.True(t, errors.Is(users.DeleteUser(ctx, "alice@example.com"), user.ErrNotFound))

	var events []string
	for range 5 {
		select {
		case e := <-published:
			events = append(events, e.Type+" "+e.Subject+" by "+e.By)
		case <-time.After(time.Second):
			t.Fatalf("events so far: %v", events)
		}
	}
	require.Equal(t, []string{
		"user.updated users/alice@example.com by admin",
		"rolebinding.updated rolebindings/family-writes by admin",
		"token.created tokens/" + tok.ID + " by admin",
		"token.created tokens/" + old.ID + " by admin",
		"user.deleted users/alice@example.com by admin",
	}, events)
}

func TestGrantingThroughPostgres(t *testing.T) {
	users := user.NewService(openUserDB(t), &event.Bus{})
	users.AllAuthorizers = user.Authorizers{user.Privileged("admins"), users}
	trusted := context.Background()
	as := func(name string, groups ...string) context.Context {
		return user.WithUser(trusted, user.User{Name: name, Groups: append(groups, user.Authenticated)})
	}

	manage := user.Rule{Verbs: []string{"get", "list", "create", "update", "delete"}, Resources: []string{"users", "roles", "rolebindings", "tokens"}}
	edit := user.Rule{Verbs: []string{"create"}, Resources: []string{"entries/edits"}}
	require.NoError(t, users.PutRole(trusted, user.Role{Name: "manager", Rules: []user.Rule{manage, edit}}))
	require.NoError(t, users.PutBinding(trusted, user.Binding{Name: "managers", Role: "manager", Subjects: []user.Subject{{Kind: "group", Name: "managers"}}}))
	require.NoError(t, users.PutUser(trusted, user.User{Name: "bob"}))
	mgr := as("mia", "managers")

	require.NoError(t, users.PutRole(mgr, user.Role{Name: "editor", Rules: []user.Rule{edit}}), "has it")
	require.ErrorIs(t, users.PutRole(mgr, user.Role{Name: "fetcher", Rules: []user.Rule{{Verbs: []string{"create"}, Resources: []string{"providers/fetch"}}}}), user.ErrForbidden)
	require.ErrorIs(t, users.PutRole(mgr, user.Role{Name: "all", Rules: []user.Rule{{Verbs: []string{"*"}, Resources: []string{"*"}}}}), user.ErrForbidden)
	require.NoError(t, users.PutBinding(mgr, user.Binding{Name: "bob-edits", Role: "editor", Subjects: []user.Subject{{Kind: "user", Name: "bob"}}}))
	require.ErrorIs(t, users.PutBinding(mgr, user.Binding{Name: "bob-writes", Role: "writer", Subjects: []user.Subject{{Kind: "user", Name: "bob"}}}), user.ErrForbidden)

	require.NoError(t, users.PutUser(mgr, user.User{Name: "bob", Groups: []string{"managers"}}), "mia is in managers")
	require.ErrorIs(t, users.PutUser(mgr, user.User{Name: "bob", Groups: []string{"managers", "admins"}}), user.ErrForbidden)
	require.NoError(t, users.PutUser(as("root", "admins"), user.User{Name: "bob", Groups: []string{"managers", "admins"}}), "admins may do anything")
	require.NoError(t, users.PutUser(mgr, user.User{Name: "bob", Groups: []string{"managers", "admins"}}), "keeping a group bob already has")

	require.NoError(t, users.PutUser(trusted, user.User{Name: "mia"}))
	_, _, err := users.CreateToken(mgr, "mia", time.Time{})
	require.NoError(t, err, "a token for yourself")
	_, _, err = users.CreateToken(mgr, "bob", time.Time{})
	require.ErrorIs(t, err, user.ErrForbidden, "a token for bob would act as bob")

	require.NoError(t, users.PutRole(trusted, user.Role{Name: "granter", Rules: []user.Rule{{Verbs: []string{"escalate", "bind"}, Resources: []string{"roles"}}}}))
	require.NoError(t, users.PutBinding(trusted, user.Binding{Name: "mia-grants", Role: "granter", Subjects: []user.Subject{{Kind: "user", Name: "mia"}}}))
	require.NoError(t, users.PutBinding(mgr, user.Binding{Name: "bob-writes", Role: "writer", Subjects: []user.Subject{{Kind: "user", Name: "bob"}}}))

	ok, err := users.ReviewAccess(as("bob"), "create", "entries/edits", "tmdb:tv:1")
	require.NoError(t, err)
	require.True(t, ok)
	ok, _ = users.ReviewAccess(as("nobody"), "create", "entries/edits", "tmdb:tv:1")
	require.False(t, ok)
}
