-- name: DatabaseTime :one
-- Proves the database answers, for the health check.
SELECT now()::timestamptz AS now;
