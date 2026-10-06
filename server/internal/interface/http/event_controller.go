package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"server/internal/event"
	"server/internal/user"
)

type EventController struct {
	Events     *event.Bus
	Authorizer user.Authorizer
}

func (c *EventController) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/events", c.Watch)
}

func (c *EventController) Watch(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("watch") != "true" {
		RespondError(w, r, fmt.Errorf("%w: events are only watched, add ?watch=true", errBadRequest))
		return
	}
	ctx := r.Context()
	u, _ := user.FromContext(ctx)
	events, cancel, err := c.Events.Subscribe(r.Header.Get("Last-Event-ID"), func(event.Event) bool { return true })
	if err != nil {
		RespondError(w, r, err)
		return
	}
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flush := http.NewResponseController(w).Flush
	_ = flush()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-events:
			if !ok {
				return
			}
			resource, name, _ := strings.Cut(e.Subject, "/")
			d, err := c.Authorizer.Authorize(ctx, user.Attributes{User: u, Verb: "get", Resource: resource, Name: name})
			if err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msg("watch: authorize")
				return
			}
			if d != user.Allow {
				continue
			}
			data, err := sonic.Marshal(e)
			if err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Str("type", e.Type).Msg("watch: marshal")
				continue
			}
			if _, err := fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", e.ID, e.Type, data); err != nil || flush() != nil {
				return
			}
		}
	}
}
