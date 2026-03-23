package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewRouter(logger zerolog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(traceRequests)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(injectLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.Heartbeat("/health"))
	return r
}

func traceRequests(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "http",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
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
