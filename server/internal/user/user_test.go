package user_test

import (
	"context"
	"testing"

	"server/internal/user"
)

func TestAllowed(t *testing.T) {
	roles := []user.Role{
		{Name: "reader", Rules: []user.Rule{{Verbs: []string{"get", "list"}, Resources: []string{"entries"}}}},
		{Name: "tmdb-editor", Rules: []user.Rule{{Verbs: []string{"create"}, Resources: []string{"entries/edits"}, Names: []string{"tmdb:*"}}}},
		{Name: "lister", Rules: []user.Rule{{Verbs: []string{"*"}, Resources: []string{"*"}, Names: []string{"*"}}}},
	}
	bindings := []user.Binding{
		{Name: "readers", Role: "reader", Subjects: []user.Subject{{Kind: "group", Name: "family"}}},
		{Name: "bob-edits", Role: "tmdb-editor", Subjects: []user.Subject{{Kind: "user", Name: "bob"}}},
		{Name: "dangling", Role: "gone", Subjects: []user.Subject{{Kind: "user", Name: "carol"}}},
		{Name: "dave-all", Role: "lister", Subjects: []user.Subject{{Kind: "user", Name: "dave"}}},
	}
	alice := user.User{Name: "alice", Groups: []string{"family"}}
	bob := user.User{Name: "bob"}

	for _, c := range []struct {
		why  string
		a    user.Attributes
		want bool
	}{
		{"group member reads", user.Attributes{User: alice, Verb: "get", Resource: "entries", Name: "tmdb:tv:1"}, true},
		{"group member lists", user.Attributes{User: alice, Verb: "list", Resource: "entries"}, true},
		{"reading isn't editing", user.Attributes{User: alice, Verb: "create", Resource: "entries/edits", Name: "tmdb:tv:1"}, false},
		{"a resource isn't its subresource", user.Attributes{User: alice, Verb: "get", Resource: "entries/edits", Name: "tmdb:tv:1"}, false},
		{"name pattern matches", user.Attributes{User: bob, Verb: "create", Resource: "entries/edits", Name: "tmdb:tv:1:s1e5"}, true},
		{"name pattern doesn't", user.Attributes{User: bob, Verb: "create", Resource: "entries/edits", Name: "tvdb:series:1"}, false},
		{"bound by name only", user.Attributes{User: user.User{Name: "bobby"}, Verb: "create", Resource: "entries/edits", Name: "tmdb:x"}, false},
		{"missing role grants nothing", user.Attributes{User: user.User{Name: "carol"}, Verb: "get", Resource: "entries", Name: "x"}, false},
		{"star name covers no name", user.Attributes{User: user.User{Name: "dave"}, Verb: "list", Resource: "users"}, true},
		{"nobody bound", user.Attributes{User: user.User{Name: "eve"}, Verb: "get", Resource: "entries", Name: "x"}, false},
	} {
		if got := user.Allowed(roles, bindings, c.a); got != c.want {
			t.Errorf("%s: got %v, want %v", c.why, got, c.want)
		}
	}
}

type decide user.Decision

func (d decide) Authorize(context.Context, user.Attributes) (user.Decision, error) {
	return user.Decision(d), nil
}

func TestAuthorizersFirstOpinionWins(t *testing.T) {
	ctx, a := context.Background(), user.Attributes{User: user.User{Name: "x", Groups: []string{"admins"}}}
	for _, c := range []struct {
		chain user.Authorizers
		want  user.Decision
	}{
		{user.Authorizers{}, user.Deny},
		{user.Authorizers{decide(user.NoOpinion)}, user.Deny},
		{user.Authorizers{decide(user.NoOpinion), decide(user.Allow)}, user.Allow},
		{user.Authorizers{decide(user.Deny), decide(user.Allow)}, user.Deny},
		{user.Authorizers{user.Privileged("admins"), decide(user.Deny)}, user.Allow},
		{user.Authorizers{user.Privileged("ops"), decide(user.NoOpinion)}, user.Deny},
	} {
		if got, _ := c.chain.Authorize(ctx, a); got != c.want {
			t.Errorf("%v: got %v, want %v", c.chain, got, c.want)
		}
	}
}

func TestAuthenticators(t *testing.T) {
	ctx := context.Background()
	chain := user.Authenticators{
		user.StaticToken{Token: "admin-secret", User: user.User{Name: "admin"}},
		user.StaticToken{Token: "other-secret", User: user.User{Name: "other"}},
		user.StaticToken{},
	}
	for token, want := range map[string]string{"admin-secret": "admin", "other-secret": "other", "nope": "", "": ""} {
		u, ok, err := chain.Authenticate(ctx, token)
		if err != nil || ok != (want != "") || u.Name != want {
			t.Errorf("%q: got %v %v %v, want %q", token, u, ok, err, want)
		}
	}
}
