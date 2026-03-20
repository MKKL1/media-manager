package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func NewRouter(logger zerolog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(zerologMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.Heartbeat("/health"))

	return r
}

func zerologMiddleware(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			reqLogger := logger.With().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("request_id", middleware.GetReqID(r.Context())).
				Str("remote_addr", r.RemoteAddr).
				Logger()

			ctx := reqLogger.WithContext(r.Context())
			next.ServeHTTP(ww, r.WithContext(ctx))

			status := ww.Status()
			lvl := zerolog.InfoLevel
			if status >= 500 {
				lvl = zerolog.ErrorLevel
			} else if status >= 400 {
				lvl = zerolog.WarnLevel
			}

			reqLogger.WithLevel(lvl).
				Int("status", status).
				Int("bytes", ww.BytesWritten()).
				Dur("duration", time.Since(start)).
				Msg("request completed")
		})
	}
}
