-- Row level security scopes every query to the signed in user.

-- name: ListPlans :many
SELECT * FROM plans
WHERE (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY created_at DESC;

-- name: GetPlan :one
SELECT * FROM plans WHERE id = $1;

-- name: CreatePlan :one
INSERT INTO plans (name, goal, notes, start_date) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdatePlan :one
UPDATE plans
SET name = $2, goal = $3, notes = $4, status = $5, start_date = $6, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeletePlan :execrows
DELETE FROM plans WHERE id = $1;

-- name: ListSessions :many
SELECT s.*, w.id AS workout_id
FROM planned_sessions s
LEFT JOIN workouts w ON w.planned_session_id = s.id
WHERE s.plan_id = ANY(sqlc.arg(plan_ids)::uuid[])
ORDER BY s.plan_id, s.position;

-- name: GetSession :one
SELECT s.*, w.id AS workout_id
FROM planned_sessions s
LEFT JOIN workouts w ON w.planned_session_id = s.id
WHERE s.id = $1;

-- name: GetSessionForUpdate :one
SELECT * FROM planned_sessions WHERE id = $1 FOR UPDATE;

-- name: MaxSessionPosition :one
SELECT coalesce(max(position), -1)::int AS position FROM planned_sessions WHERE plan_id = $1;

-- name: ShiftSessions :exec
-- Makes room for inserted sessions. The position constraint is deferred, so
-- the shift may pass through duplicates within the transaction.
UPDATE planned_sessions SET position = position + sqlc.arg(by)::int
WHERE plan_id = sqlc.arg(plan_id) AND position > sqlc.arg(after)::int;

-- name: InsertSession :exec
INSERT INTO planned_sessions (id, plan_id, position, label, week, day, intensity, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateSession :exec
UPDATE planned_sessions
SET label = $2, week = $3, day = $4, intensity = $5, notes = $6, updated_at = now()
WHERE id = $1;

-- name: SetSessionPosition :exec
UPDATE planned_sessions SET position = $2, updated_at = now() WHERE id = $1;

-- name: SetSessionStatus :exec
UPDATE planned_sessions
SET status = $2, skip_reason = $3, completed_at = $4, updated_at = now()
WHERE id = $1;

-- name: DeleteSession :execrows
DELETE FROM planned_sessions WHERE id = $1;

-- name: DeleteSessionExercises :exec
DELETE FROM planned_exercises WHERE session_id = $1;

-- name: InsertPlannedExercise :batchexec
INSERT INTO planned_exercises (id, session_id, position, exercise_id, notes, off_arm_percent)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertTarget :batchexec
INSERT INTO planned_targets (id, planned_exercise_id, position, set_type, sets, reps, weight, time, rpe_min, rpe_max, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: ListPlannedExercises :many
SELECT pe.*, e.name AS exercise_name, e.muscle_group, e.type AS exercise_type,
       e.display_type, e.single_arm
FROM planned_exercises pe
JOIN exercises e ON e.id = pe.exercise_id
WHERE pe.session_id = ANY(sqlc.arg(session_ids)::uuid[])
ORDER BY pe.session_id, pe.position;

-- name: ListTargets :many
SELECT * FROM planned_targets
WHERE planned_exercise_id = ANY(sqlc.arg(planned_exercise_ids)::uuid[])
ORDER BY planned_exercise_id, position;

-- name: ListActualSets :many
-- What was logged in the workouts of these sessions, for the planned versus
-- actual comparison.
SELECT w.planned_session_id AS session_id, we.id AS workout_exercise_id, we.position AS exercise_position,
       we.name, we.planned_exercise_id, ws.position, ws.type, ws.weight, ws.reps, ws.time, ws.rpe, ws.arm,
       ws.target_id, ws.target_seq
FROM workouts w
JOIN workout_exercises we ON we.workout_id = w.id
JOIN workout_sets ws ON ws.workout_exercise_id = we.id
WHERE w.planned_session_id = ANY(sqlc.arg(session_ids)::uuid[])
ORDER BY w.planned_session_id, we.position, ws.position;

-- name: ListQueue :many
-- Sessions still to do in active plans, in the order they are meant to be done.
SELECT s.*, p.name AS plan_name, w.id AS workout_id
FROM planned_sessions s
JOIN plans p ON p.id = s.plan_id
LEFT JOIN workouts w ON w.planned_session_id = s.id
WHERE p.status = 'active' AND s.status IN ('pending', 'in_progress')
ORDER BY p.created_at, s.position;

-- name: GetSettingValue :one
SELECT value FROM settings WHERE key = $1;
