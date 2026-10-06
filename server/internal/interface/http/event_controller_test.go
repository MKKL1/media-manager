package http

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"server/internal/event"
	"server/internal/user"
)

func TestWatchSendsOnlyWhatTheWatcherMayGet(t *testing.T) {
	bus := &event.Bus{}
	authz := authorizeFunc(func(a user.Attributes) user.Decision {
		if (a.Verb == "watch" && a.Resource == "events") || (a.Verb == "get" && a.Resource == "entries" && a.Name == "tmdb:a") {
			return user.Allow
		}
		return user.NoOpinion
	})
	r := NewRouter(zerolog.Nop(), anyToken{}, authz)
	(&EventController{Events: bus, Authorizer: authz}).RegisterRoutes(r)
	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/v1/events?watch=true", nil)
	req.Header.Set("Authorization", "Bearer alice")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}

	bus.Publish(ctx, event.Event{Type: "edit.updated", Subject: "entries/tmdb:b"})
	bus.Publish(ctx, event.Event{Type: "edit.updated", Subject: "entries/tmdb:a"})
	lines := bufio.NewScanner(resp.Body)
	for lines.Scan() {
		if data, ok := strings.CutPrefix(lines.Text(), "data: "); ok {
			if !strings.Contains(data, `"subject":"entries/tmdb:a"`) {
				t.Fatalf("got %s, want only entries/tmdb:a", data)
			}
			return
		}
	}
	t.Fatalf("stream ended: %v", lines.Err())
}
