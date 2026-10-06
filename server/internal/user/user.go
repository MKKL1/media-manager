package user

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"time"
)

var (
	ErrUnauthenticated = errors.New("not signed in")
	ErrForbidden       = errors.New("not allowed")
	ErrNotFound        = errors.New("not found")
	ErrInvalid         = errors.New("invalid")
)

type User struct {
	Name   string   `json:"name"`
	Groups []string `json:"groups"`
}

func (u User) InGroup(g string) bool { return slices.Contains(u.Groups, g) }

const Authenticated = "authenticated"

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (u User, tokenIsMine bool, err error)
}

type Authenticators []Authenticator

func (as Authenticators) Authenticate(ctx context.Context, token string) (User, bool, error) {
	for _, a := range as {
		if u, ok, err := a.Authenticate(ctx, token); err != nil || ok {
			return u, ok, err
		}
	}
	return User{}, false, nil
}

type Attributes struct {
	User     User
	Verb     string
	Resource string
	Name     string
}

type Decision int

const (
	NoOpinion Decision = iota
	Allow
	Deny
)

type Authorizer interface {
	Authorize(ctx context.Context, a Attributes) (Decision, error)
}

type Authorizers []Authorizer

func (as Authorizers) Authorize(ctx context.Context, a Attributes) (Decision, error) {
	for _, az := range as {
		if d, err := az.Authorize(ctx, a); err != nil || d != NoOpinion {
			return d, err
		}
	}
	return Deny, nil
}

type Privileged string

func (g Privileged) Authorize(_ context.Context, a Attributes) (Decision, error) {
	if a.User.InGroup(string(g)) {
		return Allow, nil
	}
	return NoOpinion, nil
}

type Rule struct {
	Verbs     []string `json:"verbs"`
	Resources []string `json:"resources"`
	Names     []string `json:"names,omitempty"`
}

func (r Rule) matches(a Attributes) bool {
	has := func(list []string, v string) bool { return slices.Contains(list, "*") || slices.Contains(list, v) }
	if !has(r.Verbs, a.Verb) || !has(r.Resources, a.Resource) {
		return false
	}
	if len(r.Names) == 0 {
		return true
	}
	return slices.ContainsFunc(r.Names, func(p string) bool { ok, _ := path.Match(p, a.Name); return ok })
}

type Role struct {
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

type Subject struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func (s Subject) is(u User) bool {
	return (s.Kind == "user" && s.Name == u.Name) || (s.Kind == "group" && u.InGroup(s.Name))
}

type Binding struct {
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	Subjects []Subject `json:"subjects"`
}

func Allowed(roles []Role, bindings []Binding, a Attributes) bool {
	for _, b := range bindings {
		if !slices.ContainsFunc(b.Subjects, func(s Subject) bool { return s.is(a.User) }) {
			continue
		}
		for _, r := range roles {
			if r.Name == b.Role && slices.ContainsFunc(r.Rules, func(rule Rule) bool { return rule.matches(a) }) {
				return true
			}
		}
	}
	return false
}

type Token struct {
	ID      string    `json:"id"`
	User    string    `json:"user"`
	Created time.Time `json:"created"`
	Expires time.Time `json:"expires,omitzero"`
}

func (t Token) expired(now time.Time) bool { return !t.Expires.IsZero() && !now.Before(t.Expires) }

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._@-]{0,62}$`)

func checkName(what, name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("%w: %s name %q: letters, digits and . _ @ -, at most 63", ErrInvalid, what, name)
	}
	return nil
}

var Verbs = []string{"get", "list", "watch", "create", "update", "delete", "escalate", "bind", "impersonate", "*"}

func (u User) check() error {
	if err := checkName("user", u.Name); err != nil {
		return err
	}
	for _, g := range u.Groups {
		if err := checkName("group", g); err != nil {
			return err
		}
	}
	return nil
}

func (r Role) check() error {
	if err := checkName("role", r.Name); err != nil {
		return err
	}
	for i, rule := range r.Rules {
		if len(rule.Verbs) == 0 || len(rule.Resources) == 0 {
			return fmt.Errorf("%w: rule %d needs verbs and resources", ErrInvalid, i)
		}
		for _, v := range rule.Verbs {
			if !slices.Contains(Verbs, v) {
				return fmt.Errorf("%w: rule %d: verb %q is not one of %v", ErrInvalid, i, v, Verbs)
			}
		}
		for _, n := range rule.Names {
			if _, err := path.Match(n, ""); err != nil {
				return fmt.Errorf("%w: rule %d: name pattern %q", ErrInvalid, i, n)
			}
		}
	}
	return nil
}

func (b Binding) check() error {
	if err := checkName("binding", b.Name); err != nil {
		return err
	}
	if err := checkName("role", b.Role); err != nil {
		return err
	}
	if len(b.Subjects) == 0 {
		return fmt.Errorf("%w: binding needs subjects", ErrInvalid)
	}
	for _, s := range b.Subjects {
		if s.Kind != "user" && s.Kind != "group" {
			return fmt.Errorf("%w: subject kind %q is not user or group", ErrInvalid, s.Kind)
		}
		if err := checkName(s.Kind, s.Name); err != nil {
			return err
		}
	}
	return nil
}

type ctxKey struct{}

func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}
