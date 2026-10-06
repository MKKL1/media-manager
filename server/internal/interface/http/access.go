package http

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"server/internal/event"
	"server/internal/user"
)

func access(authn user.Authenticator, authz user.Authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			u, err := signIn(r, authn)
			if errors.Is(err, user.ErrUnauthenticated) {
				w.Header().Set("WWW-Authenticate", "Bearer")
			}
			if err != nil {
				RespondError(w, r, err)
				return
			}
			if !u.InGroup(user.Authenticated) {
				u.Groups = append(slices.Clone(u.Groups), user.Authenticated)
			}
			ctx = event.WithBy(user.WithUser(ctx, u), u.Name)

			d, err := authz.Authorize(ctx, requestAttributes(r, u))
			if err == nil && d != user.Allow {
				err = user.ErrForbidden
			}
			if err != nil {
				RespondError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func signIn(r *http.Request, authn user.Authenticator) (user.User, error) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return user.User{}, user.ErrUnauthenticated
	}
	u, ok, err := authn.Authenticate(r.Context(), token)
	if err == nil && !ok {
		err = user.ErrUnauthenticated
	}
	return u, err
}

func requestAttributes(r *http.Request, u user.User) user.Attributes {
	var segs []string
	for s := range strings.SplitSeq(strings.Trim(r.URL.EscapedPath(), "/"), "/") {
		s, _ = url.PathUnescape(s)
		segs = append(segs, s)
	}
	a := user.Attributes{User: u}
	if len(segs) > 2 && segs[0] == "api" && segs[1] == "v1" {
		segs = segs[2:]
		a.Resource = segs[0]
		if len(segs) > 1 {
			a.Name = segs[1]
		}
		if len(segs) > 2 {
			a.Resource += "/" + strings.Join(segs[2:], "/")
		}
	} else {
		a.Resource = segs[0]
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		switch {
		case r.URL.Query().Get("watch") == "true":
			a.Verb = "watch"
		case a.Name == "":
			a.Verb = "list"
		default:
			a.Verb = "get"
		}
	case http.MethodPost:
		a.Verb = "create"
	case http.MethodPut, http.MethodPatch:
		a.Verb = "update"
	default:
		a.Verb = strings.ToLower(r.Method)
	}
	return a
}
