-- name: PutUser :exec
INSERT INTO users.users (name, groups) VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET groups = EXCLUDED.groups;

-- name: DeleteUser :execrows
DELETE FROM users.users WHERE name = $1;

-- name: AllUsers :many
SELECT name, groups FROM users.users ORDER BY name;

-- name: PutToken :execrows
INSERT INTO users.tokens (id, user_name, hash, created, expires)
SELECT sqlc.arg(id)::text, name, sqlc.arg(hash)::bytea, sqlc.arg(created)::timestamptz, sqlc.narg(expires)::timestamptz
FROM users.users WHERE name = sqlc.arg(user_name);

-- name: TokenByHash :one
SELECT t.id, t.user_name, t.created, t.expires, u.groups
FROM users.tokens t JOIN users.users u ON u.name = t.user_name WHERE t.hash = $1;

-- name: DeleteToken :execrows
DELETE FROM users.tokens WHERE id = $1;

-- name: AllTokens :many
SELECT id, user_name, created, expires FROM users.tokens ORDER BY user_name, created;

-- name: PutRole :exec
INSERT INTO users.roles (name, rules) VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET rules = EXCLUDED.rules;

-- name: DeleteRole :execrows
DELETE FROM users.roles WHERE name = $1;

-- name: AllRoles :many
SELECT name, rules FROM users.roles ORDER BY name;

-- name: PutBinding :exec
INSERT INTO users.role_bindings (name, role, subjects) VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET role = EXCLUDED.role, subjects = EXCLUDED.subjects;

-- name: DeleteBinding :execrows
DELETE FROM users.role_bindings WHERE name = $1;

-- name: AllBindings :many
SELECT name, role, subjects FROM users.role_bindings ORDER BY name;
