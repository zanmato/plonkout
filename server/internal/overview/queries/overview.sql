-- name: Username :one
SELECT username FROM auth.users WHERE id = $1;

-- name: Settings :many
SELECT key, value FROM settings WHERE key = ANY(sqlc.arg(keys)::text[]);

-- name: RecentWorkouts :many
SELECT w.id, w.name, w.started, w.ended, w.planned_session_id,
       coalesce(array_agg(we.name ORDER BY we.position) FILTER (WHERE we.name IS NOT NULL), '{}')::text[] AS exercises
FROM workouts w
LEFT JOIN workout_exercises we ON we.workout_id = w.id
GROUP BY w.id
ORDER BY w.started DESC
LIMIT $1;

-- name: CountWorkouts :one
SELECT count(*) FROM workouts;
