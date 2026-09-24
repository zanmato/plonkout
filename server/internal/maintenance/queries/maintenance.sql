-- Rows that can never be used again. Each delete is bounded by an index on
-- the expiry, and the auth schema has no row level security, so one pass
-- covers every user.

-- name: PruneCeremonies :execrows
DELETE FROM auth.webauthn_ceremonies WHERE expires_at < now();

-- name: PruneSessions :execrows
DELETE FROM auth.web_sessions WHERE expires_at < now() OR absolute_expires_at < now();

-- name: PrunePow :execrows
DELETE FROM auth.pow_spent WHERE expires_at < now();

-- name: PruneCodes :execrows
DELETE FROM auth.oauth_codes WHERE expires_at < now();

-- name: PruneAccessTokens :execrows
DELETE FROM auth.oauth_access_tokens WHERE expires_at < now();

-- name: PruneGrants :execrows
-- A revoked grant or one whose refresh token ran out cannot come back.
DELETE FROM auth.oauth_grants
WHERE revoked_at < now() - interval '30 days' OR refresh_expires_at < now() - interval '30 days';

-- name: PruneClients :execrows
-- A dynamically registered client that never got a grant within a week was
-- abandoned, and open registration would otherwise grow without bound.
DELETE FROM auth.oauth_clients c
WHERE c.created_at < now() - interval '7 days'
  AND NOT EXISTS (SELECT 1 FROM auth.oauth_grants g WHERE g.client_id = c.client_id);
