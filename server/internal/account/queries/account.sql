-- The auth schema has no row level security, because it is read before anyone
-- is known. These queries therefore always name the user they mean.

-- name: CreateUser :one
INSERT INTO auth.users (id, username, webauthn_handle)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUser :one
SELECT * FROM auth.users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM auth.users WHERE username = $1;

-- name: GetUserByHandle :one
SELECT * FROM auth.users WHERE webauthn_handle = $1;

-- name: UsernameTaken :one
SELECT EXISTS (SELECT 1 FROM auth.users WHERE username = $1);

-- name: DeleteUser :exec
DELETE FROM auth.users WHERE id = $1;

-- name: CreatePasskey :exec
INSERT INTO auth.passkeys (
  credential_id, user_id, public_key, attestation_type, aaguid, sign_count,
  transports, backup_eligible, backup_state, name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: ListPasskeys :many
SELECT * FROM auth.passkeys WHERE user_id = $1 ORDER BY created_at;

-- name: CountPasskeys :one
SELECT count(*) FROM auth.passkeys WHERE user_id = $1;

-- name: TouchPasskey :exec
UPDATE auth.passkeys
SET sign_count = $3, backup_state = $4, last_used_at = now()
WHERE credential_id = $1 AND user_id = $2;

-- name: RenamePasskey :execrows
UPDATE auth.passkeys SET name = $3 WHERE credential_id = $1 AND user_id = $2;

-- name: DeletePasskey :execrows
DELETE FROM auth.passkeys WHERE credential_id = $1 AND user_id = $2;

-- name: CreateCeremony :one
INSERT INTO auth.webauthn_ceremonies (kind, user_id, username, webauthn_handle, session_data, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: TakeCeremony :one
-- Consumes a ceremony, so finishing one twice is impossible.
DELETE FROM auth.webauthn_ceremonies
WHERE id = $1 AND kind = $2 AND expires_at > now()
RETURNING *;

-- name: CreateRecoveryCode :exec
INSERT INTO auth.recovery_codes (user_id, code_hash) VALUES ($1, $2);

-- name: DeleteRecoveryCodes :exec
DELETE FROM auth.recovery_codes WHERE user_id = $1;

-- name: RecoveryCodeUsable :one
SELECT EXISTS (
  SELECT 1 FROM auth.recovery_codes WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL
);

-- name: UseRecoveryCode :execrows
UPDATE auth.recovery_codes SET used_at = now()
WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL;

-- name: CountUnusedRecoveryCodes :one
SELECT count(*) FROM auth.recovery_codes WHERE user_id = $1 AND used_at IS NULL;

-- name: CreateSession :exec
INSERT INTO auth.web_sessions (token_hash, user_id, user_agent, expires_at, absolute_expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetSession :one
SELECT * FROM auth.web_sessions WHERE token_hash = $1 AND expires_at > now();

-- name: ExtendSession :exec
UPDATE auth.web_sessions
SET last_seen_at = now(), expires_at = least(sqlc.arg(expires_at)::timestamptz, absolute_expires_at)
WHERE token_hash = sqlc.arg(token_hash);

-- name: DeleteSession :exec
DELETE FROM auth.web_sessions WHERE token_hash = $1;

-- name: SpendPow :execrows
-- Records a redeemed proof of work salt. Zero rows means it was spent before.
INSERT INTO auth.pow_spent (salt_hash, expires_at) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: SeedExercises :exec
-- Runs as the new user, so user_id comes from the connection.
INSERT INTO exercises (name, muscle_group, single_arm, type, display_type)
SELECT name, muscle_group, single_arm, type, display_type
FROM jsonb_to_recordset(sqlc.arg(exercises)::jsonb)
  AS e(name text, muscle_group text, single_arm boolean, type text, display_type text);
