-- Row level security scopes every query to the signed in user.

-- name: ListExercises :many
SELECT * FROM exercises ORDER BY muscle_group, name;

-- name: GetExercise :one
SELECT * FROM exercises WHERE id = $1;

-- name: FindExerciseByName :one
SELECT * FROM exercises WHERE lower(name) = lower($1);

-- name: CreateExercise :one
INSERT INTO exercises (name, muscle_group, type, display_type, single_arm)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateExercise :one
UPDATE exercises
SET name = $2, muscle_group = $3, type = $4, display_type = $5, single_arm = $6,
    archived = $7, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: RenameExerciseSnapshots :exec
-- History is keyed by name, so a rename carries it along.
UPDATE workout_exercises SET name = sqlc.arg(new_name)
WHERE exercise_id = sqlc.arg(exercise_id) OR lower(name) = lower(sqlc.arg(old_name));

-- name: RenameBlockPeriodizationKey :exec
-- Block periodization state is a settings document keyed by exercise name.
UPDATE settings
SET value = (value - sqlc.arg(old_name)::text)
    || jsonb_build_object(sqlc.arg(new_name)::text, value -> sqlc.arg(old_name)::text),
    updated_at = now()
WHERE key = 'blockPeriodization_exercises' AND value ? sqlc.arg(old_name)::text;

-- name: SuggestExercises :many
-- Closest names first, for an unknown name typed by a person or a model.
SELECT name FROM exercises
WHERE similarity(name, sqlc.arg(name)::text) > 0.2
ORDER BY similarity(name, sqlc.arg(name)::text) DESC
LIMIT 5;
