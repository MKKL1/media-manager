package user

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"slices"
	"time"

	"server/internal/event"
)

type Store interface {
	PutUser(ctx context.Context, u User) error
	DeleteUser(ctx context.Context, name string) error
	Users(ctx context.Context) ([]User, error)

	PutToken(ctx context.Context, t Token, hash []byte) error

	TokenByHash(ctx context.Context, hash []byte) (Token, User, error)
	DeleteToken(ctx context.Context, id string) error
	Tokens(ctx context.Context) ([]Token, error)

	PutRole(ctx context.Context, r Role) error
	DeleteRole(ctx context.Context, name string) error
	Roles(ctx context.Context) ([]Role, error)

	PutBinding(ctx context.Context, b Binding) error
	DeleteBinding(ctx context.Context, name string) error
	Bindings(ctx context.Context) ([]Binding, error)
}

type Service struct {
	store  Store
	events *event.Bus

	AllAuthorizers Authorizer
}

func NewService(store Store, events *event.Bus) *Service {
	return &Service{store: store, events: events}
}

var (
	_ Authenticator = (*Service)(nil)
	_ Authorizer    = (*Service)(nil)
)

func hashToken(token string) []byte { h := sha256.Sum256([]byte(token)); return h[:] }

func (s *Service) Authenticate(ctx context.Context, token string) (User, bool, error) {
	t, u, err := s.store.TokenByHash(ctx, hashToken(token))
	if errors.Is(err, ErrNotFound) || (err == nil && t.expired(time.Now())) {
		return User{}, false, nil
	}
	return u, err == nil, err
}

func (s *Service) Authorize(ctx context.Context, a Attributes) (Decision, error) {
	roles, err := s.store.Roles(ctx)
	if err != nil {
		return Deny, err
	}
	bindings, err := s.store.Bindings(ctx)
	if err != nil {
		return Deny, err
	}
	if Allowed(roles, bindings, a) {
		return Allow, nil
	}
	return NoOpinion, nil
}

func (s *Service) ReviewAccess(ctx context.Context, verb, resource, name string) (allowed bool, err error) {
	u, signedIn := FromContext(ctx)
	if !signedIn {
		return true, nil
	}
	all := s.AllAuthorizers
	if all == nil {
		all = s
	}
	d, err := all.Authorize(ctx, Attributes{User: u, Verb: verb, Resource: resource, Name: name})
	return d == Allow, err
}

func (s *Service) mayGrant(ctx context.Context, verb, role string, rules []Rule) error {
	if ok, err := s.ReviewAccess(ctx, verb, "roles", role); err != nil || ok {
		return err
	}
	for _, r := range rules {
		names := r.Names
		if len(names) == 0 {
			names = []string{""}
		}
		for _, v := range r.Verbs {
			for _, res := range r.Resources {
				for _, n := range names {
					ok, err := s.ReviewAccess(ctx, v, res, n)
					if err != nil {
						return err
					}
					if !ok {
						return fmt.Errorf("%w: granting %s on %s %q needs you to have it, or %s on roles/%s", ErrForbidden, v, res, n, verb, role)
					}
				}
			}
		}
	}
	return nil
}

func (s *Service) publish(ctx context.Context, typ, subject string, data any) {
	by, _ := FromContext(ctx)
	s.events.Publish(ctx, event.Event{Type: typ, Subject: subject, By: by.Name, Data: data})
}

func (s *Service) done(ctx context.Context, err error, typ, subject string, data any) error {
	if err == nil {
		s.publish(ctx, typ, subject, data)
	}
	return err
}

func (s *Service) PutUser(ctx context.Context, u User) error {
	if u.Groups == nil {
		u.Groups = []string{}
	}
	if err := u.check(); err != nil {
		return err
	}
	users, err := s.store.Users(ctx)
	if err != nil {
		return err
	}
	var had []string
	if i := slices.IndexFunc(users, func(o User) bool { return o.Name == u.Name }); i >= 0 {
		had = users[i].Groups
	}
	asking, _ := FromContext(ctx)
	for _, g := range u.Groups {
		if slices.Contains(had, g) || asking.InGroup(g) {
			continue
		}
		if ok, err := s.ReviewAccess(ctx, "bind", "groups", g); err != nil || !ok {
			return cmp.Or(err, fmt.Errorf("%w: putting %s in group %s needs you in it, or bind on groups/%s", ErrForbidden, u.Name, g, g))
		}
	}
	return s.done(ctx, s.store.PutUser(ctx, u), "user.updated", "users/"+u.Name, u)
}

func (s *Service) DeleteUser(ctx context.Context, name string) error {
	return s.done(ctx, s.store.DeleteUser(ctx, name), "user.deleted", "users/"+name, nil)
}

func (s *Service) Users(ctx context.Context) ([]User, error) { return s.store.Users(ctx) }

func (s *Service) CreateToken(ctx context.Context, user string, expires time.Time) (Token, string, error) {
	if asking, _ := FromContext(ctx); asking.Name != user {
		if ok, err := s.ReviewAccess(ctx, "impersonate", "users", user); err != nil || !ok {
			return Token{}, "", cmp.Or(err, fmt.Errorf("%w: a token for %s needs impersonate on users/%s", ErrForbidden, user, user))
		}
	}
	secret := rand.Text()
	t := Token{ID: rand.Text()[:10], User: user, Created: time.Now().UTC().Truncate(time.Microsecond), Expires: expires.UTC()}
	if err := s.done(ctx, s.store.PutToken(ctx, t, hashToken(secret)), "token.created", "tokens/"+t.ID, t); err != nil {
		return Token{}, "", err
	}
	return t, secret, nil
}

func (s *Service) DeleteToken(ctx context.Context, id string) error {
	return s.done(ctx, s.store.DeleteToken(ctx, id), "token.deleted", "tokens/"+id, nil)
}

func (s *Service) Tokens(ctx context.Context) ([]Token, error) { return s.store.Tokens(ctx) }

func (s *Service) PutRole(ctx context.Context, r Role) error {
	if err := r.check(); err != nil {
		return err
	}
	if err := s.mayGrant(ctx, "escalate", r.Name, r.Rules); err != nil {
		return err
	}
	return s.done(ctx, s.store.PutRole(ctx, r), "role.updated", "roles/"+r.Name, r)
}

func (s *Service) DeleteRole(ctx context.Context, name string) error {
	return s.done(ctx, s.store.DeleteRole(ctx, name), "role.deleted", "roles/"+name, nil)
}

func (s *Service) Roles(ctx context.Context) ([]Role, error) { return s.store.Roles(ctx) }

func (s *Service) PutBinding(ctx context.Context, b Binding) error {
	if err := b.check(); err != nil {
		return err
	}
	roles, err := s.store.Roles(ctx)
	if err != nil {
		return err
	}
	var rules []Rule
	if i := slices.IndexFunc(roles, func(r Role) bool { return r.Name == b.Role }); i >= 0 {
		rules = roles[i].Rules
	}
	if err := s.mayGrant(ctx, "bind", b.Role, rules); err != nil {
		return err
	}
	return s.done(ctx, s.store.PutBinding(ctx, b), "rolebinding.updated", "rolebindings/"+b.Name, b)
}

func (s *Service) DeleteBinding(ctx context.Context, name string) error {
	return s.done(ctx, s.store.DeleteBinding(ctx, name), "rolebinding.deleted", "rolebindings/"+name, nil)
}

func (s *Service) Bindings(ctx context.Context) ([]Binding, error) { return s.store.Bindings(ctx) }

type StaticToken struct {
	Token string
	User  User
}

func (s StaticToken) Authenticate(_ context.Context, token string) (User, bool, error) {
	ok := s.Token != "" && subtle.ConstantTimeCompare([]byte(s.Token), []byte(token)) == 1
	if !ok {
		return User{}, false, nil
	}
	return s.User, true, nil
}
