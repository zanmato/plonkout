-- name: ListSettings :many
SELECT key, value FROM settings ORDER BY key;

-- name: PutSetting :exec
INSERT INTO settings (key, value) VALUES ($1, $2)
ON CONFLICT (user_id, key) DO UPDATE SET value = excluded.value, updated_at = now();

-- name: DeleteSetting :exec
DELETE FROM settings WHERE key = $1;
