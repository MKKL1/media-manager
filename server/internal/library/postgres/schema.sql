CREATE SCHEMA IF NOT EXISTS library;
CREATE TABLE IF NOT EXISTS library.fetched (
    provider       text   NOT NULL,
    arrangement    text   NOT NULL,
    top_id         text   NOT NULL,
    first_fetched  bigint GENERATED ALWAYS AS IDENTITY,
    content        jsonb  NOT NULL,
    PRIMARY KEY (provider, arrangement, top_id)
);
CREATE TABLE IF NOT EXISTS library.mappings (
    provider    text   NOT NULL,
    own_id      text   NOT NULL,
    first_put   bigint GENERATED ALWAYS AS IDENTITY,
    content     jsonb  NOT NULL,
    PRIMARY KEY (provider, own_id)
);
CREATE SEQUENCE IF NOT EXISTS library.edit_order;
CREATE TABLE IF NOT EXISTS library.value_edits (
    provider text NOT NULL, entry_id text NOT NULL, field text NOT NULL,
    value text NOT NULL CHECK (value <> ''),
    saved bigint NOT NULL DEFAULT nextval('library.edit_order'),
    PRIMARY KEY (provider, entry_id, field)
);
CREATE TABLE IF NOT EXISTS library.adds_to_groups (
    provider text NOT NULL, group_id text NOT NULL, entry_id text NOT NULL,
    saved bigint NOT NULL DEFAULT nextval('library.edit_order'),
    PRIMARY KEY (provider, entry_id, group_id)
);
CREATE TABLE IF NOT EXISTS library.links (
    a_provider text COLLATE "C" NOT NULL, a_id text COLLATE "C" NOT NULL,
    b_provider text COLLATE "C" NOT NULL, b_id text COLLATE "C" NOT NULL,
    kind text NOT NULL CHECK (kind IN ('same', 'not same', 'contains', 'part of')),
    why text[] NOT NULL DEFAULT '{}',
    saved bigint NOT NULL DEFAULT nextval('library.edit_order'),
    PRIMARY KEY (a_provider, a_id, b_provider, b_id),
    CHECK ((a_provider, a_id) < (b_provider, b_id))
);
CREATE TABLE IF NOT EXISTS library.choices (
    provider text NOT NULL, entry_id text NOT NULL, chosen boolean NOT NULL,
    saved bigint NOT NULL DEFAULT nextval('library.edit_order'),
    PRIMARY KEY (provider, entry_id)
);
