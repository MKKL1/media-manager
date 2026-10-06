package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"server/internal/user"
)

func NewRouter(logger zerolog.Logger, authn user.Authenticator, authz user.Authorizer) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(injectLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(timeoutUnlessWatching(30 * time.Second))
	r.Use(middleware.Heartbeat("/health"))
	r.Use(access(authn, authz))
	return r
}

func timeoutUnlessWatching(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		limited := middleware.Timeout(d)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("watch") == "true" {
				next.ServeHTTP(w, r)
				return
			}
			limited.ServeHTTP(w, r)
		})
	}
}

func injectLogger(base zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			log := base.With().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("request_id", middleware.GetReqID(r.Context())).
				Logger()

			ctx := log.WithContext(r.Context())
			next.ServeHTTP(ww, r.WithContext(ctx))

			status := ww.Status()
			lvl := zerolog.InfoLevel
			if status >= 500 {
				lvl = zerolog.ErrorLevel
			} else if status >= 400 {
				lvl = zerolog.WarnLevel
			}
			log.WithLevel(lvl).
				Int("status", status).
				Dur("duration", time.Since(start)).
				Msg("request")
		})
	}
}
