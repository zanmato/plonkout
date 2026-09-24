-- name: ListTemplates :many
SELECT * FROM templates ORDER BY created_at DESC;

-- name: GetTemplate :one
SELECT * FROM templates WHERE id = $1;

-- name: CreateTemplate :one
INSERT INTO templates (name, doc) VALUES ($1, $2) RETURNING *;

-- name: UpdateTemplate :one
UPDATE templates SET name = $2, doc = $3, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteTemplate :execrows
DELETE FROM templates WHERE id = $1;
