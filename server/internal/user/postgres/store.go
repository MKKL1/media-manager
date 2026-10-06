package postgres

//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"server/internal/user"
	"server/internal/user/postgres/queries"
)

//go:embed schema.sql
var schema string

func Migrate(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, schema); err != nil {
		return fmt.Errorf("migrate users: %w", err)
	}
	return nil
}

type Store struct{ q *queries.Queries }

var _ user.Store = (*Store)(nil)

func New(db *pgxpool.Pool) *Store { return &Store{queries.New(db)} }

func deleted(n int64, err error, what string) error {
	if err != nil {
		return fmt.Errorf("delete %s: %w", what, err)
	}
	if n == 0 {
		return fmt.Errorf("%s: %w", what, user.ErrNotFound)
	}
	return nil
}

func (s *Store) PutUser(ctx context.Context, u user.User) error {
	if err := s.q.PutUser(ctx, queries.PutUserParams{Name: u.Name, Groups: groupsOf(u.Groups)}); err != nil {
		return fmt.Errorf("put user %s: %w", u.Name, err)
	}
	return nil
}

func groupsOf(groups []string) []string {
	if groups == nil {
		return []string{}
	}
	return groups
}

func (s *Store) DeleteUser(ctx context.Context, name string) error {
	n, err := s.q.DeleteUser(ctx, name)
	return deleted(n, err, "user "+name)
}

func (s *Store) Users(ctx context.Context) ([]user.User, error) {
	rows, err := s.q.AllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("users: %w", err)
	}
	users := make([]user.User, len(rows))
	for i, r := range rows {
		users[i] = user.User{Name: r.Name, Groups: r.Groups}
	}
	return users, nil
}

func expiresOf(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func tokenOf(id, userName string, created time.Time, expires *time.Time) user.Token {
	t := user.Token{ID: id, User: userName, Created: created.UTC()}
	if expires != nil {
		t.Expires = expires.UTC()
	}
	return t
}

func (s *Store) PutToken(ctx context.Context, t user.Token, hash []byte) error {
	n, err := s.q.PutToken(ctx, queries.PutTokenParams{ID: t.ID, Hash: hash, Created: t.Created, Expires: expiresOf(t.Expires), UserName: t.User})
	if err != nil {
		return fmt.Errorf("put token for %s: %w", t.User, err)
	}
	if n == 0 {
		return fmt.Errorf("user %s: %w", t.User, user.ErrNotFound)
	}
	return nil
}

func (s *Store) TokenByHash(ctx context.Context, hash []byte) (user.Token, user.User, error) {
	r, err := s.q.TokenByHash(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.Token{}, user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.Token{}, user.User{}, fmt.Errorf("token by hash: %w", err)
	}
	return tokenOf(r.ID, r.UserName, r.Created, r.Expires), user.User{Name: r.UserName, Groups: r.Groups}, nil
}

func (s *Store) DeleteToken(ctx context.Context, id string) error {
	n, err := s.q.DeleteToken(ctx, id)
	return deleted(n, err, "token "+id)
}

func (s *Store) Tokens(ctx context.Context) ([]user.Token, error) {
	rows, err := s.q.AllTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("tokens: %w", err)
	}
	tokens := make([]user.Token, len(rows))
	for i, r := range rows {
		tokens[i] = tokenOf(r.ID, r.UserName, r.Created, r.Expires)
	}
	return tokens, nil
}

func (s *Store) PutRole(ctx context.Context, r user.Role) error {
	rules, err := json.Marshal(r.Rules)
	if err == nil {
		err = s.q.PutRole(ctx, queries.PutRoleParams{Name: r.Name, Rules: rules})
	}
	if err != nil {
		return fmt.Errorf("put role %s: %w", r.Name, err)
	}
	return nil
}

func (s *Store) DeleteRole(ctx context.Context, name string) error {
	n, err := s.q.DeleteRole(ctx, name)
	return deleted(n, err, "role "+name)
}

func (s *Store) Roles(ctx context.Context) ([]user.Role, error) {
	rows, err := s.q.AllRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles: %w", err)
	}
	roles := make([]user.Role, len(rows))
	for i, r := range rows {
		roles[i].Name = r.Name
		if err := json.Unmarshal(r.Rules, &roles[i].Rules); err != nil {
			return nil, fmt.Errorf("role %s: %w", r.Name, err)
		}
	}
	return roles, nil
}

func (s *Store) PutBinding(ctx context.Context, b user.Binding) error {
	subjects, err := json.Marshal(b.Subjects)
	if err == nil {
		err = s.q.PutBinding(ctx, queries.PutBindingParams{Name: b.Name, Role: b.Role, Subjects: subjects})
	}
	if err != nil {
		return fmt.Errorf("put binding %s: %w", b.Name, err)
	}
	return nil
}

func (s *Store) DeleteBinding(ctx context.Context, name string) error {
	n, err := s.q.DeleteBinding(ctx, name)
	return deleted(n, err, "binding "+name)
}

func (s *Store) Bindings(ctx context.Context) ([]user.Binding, error) {
	rows, err := s.q.AllBindings(ctx)
	if err != nil {
		return nil, fmt.Errorf("bindings: %w", err)
	}
	bindings := make([]user.Binding, len(rows))
	for i, r := range rows {
		bindings[i].Name, bindings[i].Role = r.Name, r.Role
		if err := json.Unmarshal(r.Subjects, &bindings[i].Subjects); err != nil {
			return nil, fmt.Errorf("binding %s: %w", r.Name, err)
		}
	}
	return bindings, nil
}
