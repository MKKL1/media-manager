package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"server/internal/user"
)

type anyToken struct{}

func (anyToken) Authenticate(_ context.Context, token string) (user.User, bool, error) {
	return user.User{Name: token}, true, nil
}

type allowAll struct{}

func (allowAll) Authorize(context.Context, user.Attributes) (user.Decision, error) {
	return user.Allow, nil
}

func TestRequestAttributes(t *testing.T) {
	for _, c := range []struct{ method, path, verb, resource, name string }{
		{"GET", "/api/v1/library", "list", "library", ""},
		{"GET", "/api/v1/entries/tmdb:tv:209867", "get", "entries", "tmdb:tv:209867"},
		{"POST", "/api/v1/entries/tmdb:tv:1/edits", "create", "entries/edits", "tmdb:tv:1"},
		{"GET", "/api/v1/entries/tmdb:tv:1/view?arrangement=dvd", "get", "entries/view", "tmdb:tv:1"},
		{"GET", "/api/v1/providers/tmdb/search?q=x", "get", "providers/search", "tmdb"},
		{"POST", "/api/v1/tokens", "create", "tokens", ""},
		{"PUT", "/api/v1/roles/reader", "update", "roles", "reader"},
		{"DELETE", "/api/v1/rolebindings/basic", "delete", "rolebindings", "basic"},
		{"GET", "/api/v1/entries/a%2Fb", "get", "entries", "a/b"},
		{"GET", "/diag/queues", "list", "diag", ""},
		{"GET", "/api/v1/events?watch=true", "watch", "events", ""},
	} {
		got := requestAttributes(httptest.NewRequest(c.method, c.path, nil), user.User{})
		if got.Verb != c.verb || got.Resource != c.resource || got.Name != c.name {
			t.Errorf("%s %s: got %s %s %q", c.method, c.path, got.Verb, got.Resource, got.Name)
		}
	}
}

func TestAccess(t *testing.T) {
	authz := user.Authorizers{authorizeFunc(func(a user.Attributes) user.Decision {
		if a.User.Name == "reader" && a.Verb == "get" && a.Resource == "entries" && a.User.InGroup(user.Authenticated) {
			return user.Allow
		}
		return user.NoOpinion
	})}
	r := NewRouter(zerolog.Nop(), user.Authenticators{user.StaticToken{Token: "reader", User: user.User{Name: "reader"}}}, authz)
	r.Get("/api/v1/entries/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })

	for _, c := range []struct {
		auth, path string
		want       int
	}{
		{"", "/api/v1/entries/x", http.StatusUnauthorized},
		{"Basic reader", "/api/v1/entries/x", http.StatusUnauthorized},
		{"Bearer wrong", "/api/v1/entries/x", http.StatusUnauthorized},
		{"Bearer reader", "/api/v1/entries/x", http.StatusTeapot},
		{"Bearer reader", "/api/v1/entries", http.StatusForbidden},
		{"", "/health", http.StatusOK},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", c.path, nil)
		if c.auth != "" {
			req.Header.Set("Authorization", c.auth)
		}
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("%q %s: %d, want %d", c.auth, c.path, w.Code, c.want)
		}
	}
}

type authorizeFunc func(user.Attributes) user.Decision

func (f authorizeFunc) Authorize(_ context.Context, a user.Attributes) (user.Decision, error) {
	return f(a), nil
}
