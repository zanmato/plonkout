-- Row level security scopes every query to the signed in user.

-- name: EntriesBetween :many
SELECT * FROM food_entries
WHERE day BETWEEN sqlc.arg(from_day) AND sqlc.arg(to_day)
ORDER BY day, created_at, id;

-- name: GetEntry :one
SELECT * FROM food_entries WHERE id = $1;

-- name: CreateEntry :one
INSERT INTO food_entries (day, meal, food_id, lmv_number, name, grams, amount, kcal, protein, carbs, fat,
                          fiber, sugars, saturated_fat, salt, logged_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: UpdateEntry :one
UPDATE food_entries SET day = $2, meal = $3, grams = $4, amount = $5
WHERE id = $1
RETURNING *;

-- name: DeleteEntry :execrows
DELETE FROM food_entries WHERE id = $1;

-- name: ActivitiesBetween :many
SELECT * FROM activities
WHERE day BETWEEN sqlc.arg(from_day) AND sqlc.arg(to_day)
ORDER BY day, created_at, id;

-- name: CreateActivity :one
INSERT INTO activities (day, label, kcal, workout_id, logged_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteActivity :execrows
DELETE FROM activities WHERE id = $1;

-- name: WeightsBetween :many
SELECT * FROM body_weights
WHERE day BETWEEN sqlc.arg(from_day) AND sqlc.arg(to_day)
ORDER BY day;

-- name: SaveWeight :one
INSERT INTO body_weights (day, weight) VALUES ($1, $2)
ON CONFLICT (user_id, day) DO UPDATE SET weight = excluded.weight, updated_at = now()
RETURNING *;

-- name: DeleteWeight :execrows
DELETE FROM body_weights WHERE day = $1;

-- name: GetGoal :one
SELECT * FROM nutrition_goals;

-- name: SaveGoal :one
INSERT INTO nutrition_goals (direction, kcal, protein, carbs, fat, add_activities, target_weight,
                             weekly_change, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO UPDATE
SET direction = excluded.direction, kcal = excluded.kcal, protein = excluded.protein,
    carbs = excluded.carbs, fat = excluded.fat, add_activities = excluded.add_activities,
    target_weight = excluded.target_weight, weekly_change = excluded.weekly_change,
    notes = excluded.notes, updated_at = now()
RETURNING *;
