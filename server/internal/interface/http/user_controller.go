package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"server/internal/user"
)

type UserController struct {
	Users *user.Service
}

func (c *UserController) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/whoami", c.WhoAmI)
	r.Post("/api/v1/accessreviews", c.ReviewAccess)

	r.Get("/api/v1/users", list(c.Users.Users))
	r.Put("/api/v1/users/{name}", c.PutUser)
	r.Delete("/api/v1/users/{name}", remove(c.Users.DeleteUser))

	r.Get("/api/v1/tokens", list(c.Users.Tokens))
	r.Post("/api/v1/tokens", c.CreateToken)
	r.Delete("/api/v1/tokens/{name}", remove(c.Users.DeleteToken))

	r.Get("/api/v1/roles", list(c.Users.Roles))
	r.Put("/api/v1/roles/{name}", c.PutRole)
	r.Delete("/api/v1/roles/{name}", remove(c.Users.DeleteRole))

	r.Get("/api/v1/rolebindings", list(c.Users.Bindings))
	r.Put("/api/v1/rolebindings/{name}", c.PutBinding)
	r.Delete("/api/v1/rolebindings/{name}", remove(c.Users.DeleteBinding))
}

func list[T any](all func(context.Context) ([]T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := all(r.Context())
		if err != nil {
			RespondError(w, r, err)
			return
		}
		_ = writeJSON(w, http.StatusOK, map[string]any{"data": items})
	}
}

func remove(del func(ctx context.Context, name string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := del(r.Context(), chi.URLParam(r, "name")); err != nil {
			RespondError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func done(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		RespondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *UserController) WhoAmI(w http.ResponseWriter, r *http.Request) {
	u, _ := user.FromContext(r.Context())
	_ = writeJSON(w, http.StatusOK, map[string]any{"data": u})
}

func (c *UserController) ReviewAccess(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Verb     string `json:"verb" validate:"required"`
		Resource string `json:"resource" validate:"required"`
		Name     string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}
	allowed, err := c.Users.ReviewAccess(r.Context(), req.Verb, req.Resource, req.Name)
	if err != nil {
		RespondError(w, r, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"allowed": allowed}})
}

func (c *UserController) PutUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Groups []string `json:"groups"`
	}
	if err := decodeJSON(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}
	done(w, r, c.Users.PutUser(r.Context(), user.User{Name: chi.URLParam(r, "name"), Groups: req.Groups}))
}

func (c *UserController) CreateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User    string    `json:"user" validate:"required"`
		Expires time.Time `json:"expires"`
	}
	if err := decodeJSON(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}
	t, secret, err := c.Users.CreateToken(r.Context(), req.User, req.Expires)
	if err != nil {
		RespondError(w, r, err)
		return
	}
	_ = writeJSON(w, http.StatusCreated, map[string]any{"data": struct {
		user.Token
		Secret string `json:"token"`
	}{t, secret}})
}

func (c *UserController) PutRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rules []user.Rule `json:"rules"`
	}
	if err := decodeJSON(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}
	done(w, r, c.Users.PutRole(r.Context(), user.Role{Name: chi.URLParam(r, "name"), Rules: req.Rules}))
}

func (c *UserController) PutBinding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role     string         `json:"role"`
		Subjects []user.Subject `json:"subjects"`
	}
	if err := decodeJSON(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}
	done(w, r, c.Users.PutBinding(r.Context(), user.Binding{Name: chi.URLParam(r, "name"), Role: req.Role, Subjects: req.Subjects}))
}
