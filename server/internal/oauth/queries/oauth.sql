-- The auth schema has no row level security. These queries name the user
-- they mean wherever a user is involved.

-- name: UpsertClient :one
INSERT INTO auth.oauth_clients (client_id, kind, client_name, redirect_uris, metadata, cache_until)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (client_id) DO UPDATE
SET client_name = excluded.client_name, redirect_uris = excluded.redirect_uris,
    metadata = excluded.metadata, cache_until = excluded.cache_until
RETURNING *;

-- name: GetClient :one
SELECT * FROM auth.oauth_clients WHERE client_id = $1;

-- name: CreateCode :exec
INSERT INTO auth.oauth_codes (code_hash, client_id, user_id, redirect_uri, code_challenge, scope, resource, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: TakeCode :one
-- Consumes the code, so it works once.
DELETE FROM auth.oauth_codes WHERE code_hash = $1 AND expires_at > now() RETURNING *;

-- name: CreateGrant :one
INSERT INTO auth.oauth_grants (client_id, user_id, scope, resource, refresh_hash, refresh_expires_at, last_used_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
RETURNING *;

-- name: RotateRefreshToken :one
UPDATE auth.oauth_grants
SET previous_refresh_hash = refresh_hash, refresh_hash = sqlc.arg(new_hash),
    refresh_expires_at = sqlc.arg(refresh_expires_at), last_used_at = now()
WHERE refresh_hash = sqlc.arg(old_hash) AND revoked_at IS NULL AND refresh_expires_at > now()
RETURNING *;

-- name: RevokeGrantByPreviousRefresh :execrows
-- A rotated refresh token came back, so somebody else holds one of the two.
UPDATE auth.oauth_grants SET revoked_at = now()
WHERE previous_refresh_hash = $1 AND revoked_at IS NULL;

-- name: RevokeGrantByRefresh :exec
UPDATE auth.oauth_grants SET revoked_at = now()
WHERE (refresh_hash = $1 OR previous_refresh_hash = $1) AND revoked_at IS NULL;

-- name: CreateAccessToken :exec
INSERT INTO auth.oauth_access_tokens (token_hash, grant_id, expires_at) VALUES ($1, $2, $3);

-- name: VerifyAccessToken :one
SELECT g.id AS grant_id, g.user_id, g.scope, g.resource, t.expires_at
FROM auth.oauth_access_tokens t
JOIN auth.oauth_grants g ON g.id = t.grant_id
WHERE t.token_hash = $1 AND t.expires_at > now() AND g.revoked_at IS NULL;

-- name: RevokeAccessToken :exec
DELETE FROM auth.oauth_access_tokens WHERE token_hash = $1;

-- name: TouchGrant :exec
UPDATE auth.oauth_grants SET last_used_at = now()
WHERE id = $1 AND (last_used_at IS NULL OR last_used_at < now() - interval '5 minutes');

-- name: ListGrants :many
SELECT g.id, g.client_id, c.client_name, g.scope, g.created_at, g.last_used_at
FROM auth.oauth_grants g
JOIN auth.oauth_clients c ON c.client_id = g.client_id
WHERE g.user_id = $1 AND g.revoked_at IS NULL
ORDER BY g.created_at DESC;

-- name: RevokeGrant :execrows
UPDATE auth.oauth_grants SET revoked_at = now()
WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL;
