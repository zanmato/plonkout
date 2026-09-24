-- Row level security scopes every query to the signed in user. A workout is
-- read and written as a whole document: the workout row, its exercises and
-- their sets.

-- name: ListWorkouts :many
SELECT * FROM workouts
WHERE (sqlc.narg(from_time)::timestamptz IS NULL OR started >= sqlc.narg(from_time))
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR started < sqlc.narg(to_time))
ORDER BY started DESC;

-- name: GetWorkout :one
SELECT * FROM workouts WHERE id = $1;

-- name: LatestWorkoutByName :one
SELECT * FROM workouts
WHERE name = sqlc.arg(name)
  AND (sqlc.narg(exclude_id)::uuid IS NULL OR id <> sqlc.narg(exclude_id))
ORDER BY started DESC
LIMIT 1;

-- name: ListWorkoutExercises :many
SELECT * FROM workout_exercises
WHERE workout_id = ANY(sqlc.arg(workout_ids)::uuid[])
ORDER BY workout_id, position;

-- name: ListWorkoutSets :many
SELECT * FROM workout_sets
WHERE workout_exercise_id = ANY(sqlc.arg(workout_exercise_ids)::uuid[])
ORDER BY workout_exercise_id, position;

-- name: CreateWorkout :one
INSERT INTO workouts (id, name, started, ended, notes, planned_session_id, legacy_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, coalesce(sqlc.narg(created)::timestamptz, now()), coalesce(sqlc.narg(updated)::timestamptz, now()))
ON CONFLICT (user_id, legacy_id) DO NOTHING
RETURNING *;

-- name: UpdateWorkout :one
-- Only applies when the caller saw the current revision.
UPDATE workouts
SET name = $3, started = $4, ended = $5, notes = $6, revision = revision + 1, updated_at = now()
WHERE id = $1 AND revision = $2
RETURNING *;

-- name: DeleteWorkout :one
DELETE FROM workouts WHERE id = $1 RETURNING planned_session_id;

-- name: DeleteWorkoutExercises :exec
DELETE FROM workout_exercises WHERE workout_id = $1;

-- name: InsertWorkoutExercise :batchexec
INSERT INTO workout_exercises (
  id, workout_id, position, exercise_id, name, muscle_group, type, display_type,
  single_arm, intensity, planned_exercise_id, notes
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: InsertWorkoutSet :batchexec
INSERT INTO workout_sets (
  workout_exercise_id, position, type, weight, distance, reps, time, rpe, arm, notes, target_id, target_seq
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: ExerciseIDsByName :many
-- Links logged exercises to the exercise list when the client only sent a name.
SELECT id, lower(name)::text AS key FROM exercises WHERE lower(name) = ANY(sqlc.arg(names)::text[]);

-- name: SyncSessionStatus :exec
-- A planned session follows its workout: in progress until the workout ends,
-- completed once it has.
UPDATE planned_sessions
SET status = CASE WHEN sqlc.narg(ended)::timestamptz IS NULL THEN 'in_progress' ELSE 'completed' END,
    completed_at = sqlc.narg(ended),
    updated_at = now()
WHERE id = sqlc.arg(id) AND status <> 'skipped';

-- name: ReleaseSession :exec
-- The workout of a session was deleted, so the session is to do again.
UPDATE planned_sessions SET status = 'pending', completed_at = NULL, updated_at = now()
WHERE id = $1 AND status IN ('in_progress', 'completed');
