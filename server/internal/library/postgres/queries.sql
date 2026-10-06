-- name: PutFetched :exec
INSERT INTO library.fetched (provider, arrangement, top_id, content) VALUES ($1, $2, $3, $4)
ON CONFLICT (provider, arrangement, top_id) DO UPDATE SET content = EXCLUDED.content;

-- name: RemoveFetched :exec
DELETE FROM library.fetched WHERE provider = $1 AND arrangement = $2 AND top_id = $3;

-- name: AllFetched :many
SELECT provider, arrangement, content FROM library.fetched ORDER BY first_fetched;

-- name: PutMapping :exec
INSERT INTO library.mappings (provider, own_id, content) VALUES ($1, $2, $3)
ON CONFLICT (provider, own_id) DO UPDATE SET content = EXCLUDED.content;

-- name: RemoveMapping :exec
DELETE FROM library.mappings WHERE provider = $1 AND own_id = $2;

-- name: AllMappings :many
SELECT provider, own_id, content FROM library.mappings ORDER BY first_put;

-- name: SetValueEdit :exec
INSERT INTO library.value_edits (provider, entry_id, field, value) VALUES ($1, $2, $3, $4)
ON CONFLICT (provider, entry_id, field) DO UPDATE SET value = EXCLUDED.value, saved = EXCLUDED.saved;

-- name: ClearValueEdit :exec
DELETE FROM library.value_edits WHERE provider = $1 AND entry_id = $2 AND field = $3;

-- name: AllValueEdits :many
SELECT provider, entry_id, field, value, saved FROM library.value_edits;

-- name: AddToGroup :exec
INSERT INTO library.adds_to_groups (provider, group_id, entry_id) VALUES ($1, $2, $3)
ON CONFLICT (provider, entry_id, group_id) DO UPDATE SET saved = EXCLUDED.saved;

-- name: RemoveFromGroup :exec
DELETE FROM library.adds_to_groups WHERE provider = $1 AND group_id = $2 AND entry_id = $3;

-- name: AllAddsToGroups :many
SELECT provider, group_id, entry_id, saved FROM library.adds_to_groups;

-- name: PutLink :exec
INSERT INTO library.links (a_provider, a_id, b_provider, b_id, kind, why) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (a_provider, a_id, b_provider, b_id) DO UPDATE SET kind = EXCLUDED.kind, why = EXCLUDED.why, saved = EXCLUDED.saved;

-- name: RemoveLink :exec
DELETE FROM library.links WHERE a_provider = $1 AND a_id = $2 AND b_provider = $3 AND b_id = $4;

-- name: AllLinks :many
SELECT a_provider, a_id, b_provider, b_id, kind, why, saved FROM library.links;

-- name: SetChoice :exec
INSERT INTO library.choices (provider, entry_id, chosen) VALUES ($1, $2, $3)
ON CONFLICT (provider, entry_id) DO UPDATE SET chosen = EXCLUDED.chosen, saved = EXCLUDED.saved;

-- name: ClearChoice :exec
DELETE FROM library.choices WHERE provider = $1 AND entry_id = $2;

-- name: AllChoices :many
SELECT provider, entry_id, chosen, saved FROM library.choices;
