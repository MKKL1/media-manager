CREATE SCHEMA IF NOT EXISTS users;
CREATE TABLE IF NOT EXISTS users.users (
    name   text   PRIMARY KEY,
    groups text[] NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS users.tokens (
    id        text        PRIMARY KEY,
    user_name text        NOT NULL REFERENCES users.users (name) ON DELETE CASCADE,
    hash      bytea       NOT NULL UNIQUE,
    created   timestamptz NOT NULL,
    expires   timestamptz
);
CREATE TABLE IF NOT EXISTS users.roles (
    name  text  PRIMARY KEY,
    rules jsonb NOT NULL
);
CREATE TABLE IF NOT EXISTS users.role_bindings (
    name     text  PRIMARY KEY,
    role     text  NOT NULL,
    subjects jsonb NOT NULL
);
INSERT INTO users.roles (name, rules) VALUES
    ('reader', '[{"verbs":["get","list"],"resources":["library","entries","entries/view","providers","providers/search"]},
                 {"verbs":["watch"],"resources":["events"]}]'),
    ('writer', '[{"verbs":["get","list"],"resources":["library","entries","entries/view","providers","providers/search"]},
                 {"verbs":["watch"],"resources":["events"]},
                 {"verbs":["create"],"resources":["entries/edits","providers/fetch"]}]'),
    ('basic',  '[{"verbs":["list"],"resources":["whoami"]},{"verbs":["create"],"resources":["accessreviews"]}]')
ON CONFLICT (name) DO NOTHING;
INSERT INTO users.role_bindings (name, role, subjects) VALUES
    ('basic', 'basic', '[{"kind":"group","name":"authenticated"}]')
ON CONFLICT (name) DO NOTHING;
