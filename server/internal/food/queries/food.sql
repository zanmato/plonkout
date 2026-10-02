-- The user's own foods are scoped by row level security. The lmv schema is
-- shared by everyone.

-- name: LatestRelease :one
SELECT id, version, imported_at FROM lmv.releases ORDER BY imported_at DESC LIMIT 1;

-- name: ReleaseVocabulary :one
SELECT vocabulary FROM lmv.releases WHERE id = $1;

-- name: ReleaseImported :one
SELECT EXISTS (SELECT 1 FROM lmv.releases WHERE sha256 = $1);

-- name: UpsertLMVFood :exec
INSERT INTO lmv.foods (number, name, food_group, kcal, protein, carbs, fat, fiber, sugars,
                       saturated_fat, salt, nutrients, search_terms, retired, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, false, now())
ON CONFLICT (number) DO UPDATE
SET name = excluded.name, food_group = excluded.food_group, kcal = excluded.kcal,
    protein = excluded.protein, carbs = excluded.carbs, fat = excluded.fat, fiber = excluded.fiber,
    sugars = excluded.sugars, saturated_fat = excluded.saturated_fat, salt = excluded.salt,
    nutrients = excluded.nutrients, search_terms = excluded.search_terms, retired = false, updated_at = now();

-- name: RetireLMVFoods :execrows
-- Foods the new release no longer has.
UPDATE lmv.foods SET retired = true, updated_at = now()
WHERE NOT retired AND NOT (number = ANY(sqlc.arg(keep)::int[]));

-- name: InsertRelease :exec
INSERT INTO lmv.releases (version, sha256, foods, vocabulary) VALUES ($1, $2, $3, $4);

-- name: SearchFoods :many
-- The user's foods and Livsmedelsverket's in one list, best match first:
--   1. what the user has called exactly this (an alias), or its barcode,
--   2. foods holding every word, by Swedish stem, compound part or as part of
--      the name, before fuzzy matches that catch typos,
--   3. within those, what the user logged most in the last 90 days, then the
--      user's own foods, then the text search rank and the closest name.
-- words are the query's words escaped for LIKE, tsquery the same search for
-- to_tsquery, null when it has no words.
WITH q AS (
  SELECT to_tsquery('swedish', sqlc.narg(tsquery)::text) AS tsq
),
history AS (
  SELECT food_id, lmv_number, count(*)::int AS uses
  FROM food_entries
  WHERE day > current_date - 90
  GROUP BY food_id, lmv_number
),
aliased AS (
  SELECT food_id, lmv_number, bool_or(lower(alias) = lower(sqlc.arg(query)::text)) AS exact
  FROM food_aliases
  WHERE NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(words)::text[]) w WHERE alias NOT ILIKE '%' || w || '%')
  GROUP BY food_id, lmv_number
),
matches AS (
  SELECT 'mine'::text AS source, f.id, NULL::int AS lmv_number, f.name, f.brand,
         ''::text AS food_group, f.kcal, f.protein, f.carbs, f.fat, f.fiber, f.sugars,
         f.saturated_fat, f.salt, 1 AS source_rank,
         coalesce(f.gtin = sqlc.arg(query)::text, false) AS barcode,
         coalesce(f.search @@ q.tsq, false)
           OR NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(words)::text[]) w
                          WHERE f.name || ' ' || f.brand NOT ILIKE '%' || w || '%') AS has_words,
         coalesce(ts_rank(f.search, q.tsq), 0) AS rank
  FROM foods f, q
  WHERE f.search @@ q.tsq
     OR NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(words)::text[]) w
                    WHERE f.name || ' ' || f.brand NOT ILIKE '%' || w || '%')
     OR word_similarity(sqlc.arg(query)::text, f.name) > 0.5
     OR f.gtin = sqlc.arg(query)::text
     OR f.id IN (SELECT food_id FROM aliased)
  UNION ALL
  SELECT 'lmv', NULL::uuid, l.number, l.name, '', l.food_group, l.kcal, l.protein, l.carbs, l.fat,
         l.fiber, l.sugars, l.saturated_fat, l.salt, 2,
         false,
         coalesce(l.search @@ q.tsq, false)
           OR NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(words)::text[]) w WHERE l.name NOT ILIKE '%' || w || '%'),
         coalesce(ts_rank(l.search, q.tsq), 0)
  FROM lmv.foods l, q
  WHERE NOT l.retired
    AND (l.search @@ q.tsq
         OR NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(words)::text[]) w WHERE l.name NOT ILIKE '%' || w || '%')
         OR word_similarity(sqlc.arg(query)::text, l.name) > 0.5
         OR l.number IN (SELECT lmv_number FROM aliased))
)
-- One reference column, since a union's column takes its nullability from
-- the first branch: the food's id for the user's own, the number for lmv.
SELECT m.source, coalesce(m.id::text, m.lmv_number::text)::text AS ref, m.name, m.brand, m.food_group,
       m.kcal, m.protein, m.carbs, m.fat, m.fiber, m.sugars, m.saturated_fat, m.salt,
       coalesce(h.uses, 0)::int AS uses
FROM matches m
LEFT JOIN history h ON h.food_id = m.id OR h.lmv_number = m.lmv_number
LEFT JOIN aliased a ON a.food_id = m.id OR a.lmv_number = m.lmv_number
ORDER BY (m.barcode OR coalesce(a.exact, false)) DESC,
         (m.has_words OR a.exact IS NOT NULL) DESC,
         coalesce(h.uses, 0) DESC,
         m.source_rank,
         m.rank DESC,
         (m.name ILIKE sqlc.arg(query)::text || '%') DESC,
         word_similarity(sqlc.arg(query)::text, m.name) DESC,
         length(m.name), m.name
LIMIT sqlc.arg(max_results);

-- name: SaveAlias :exec
-- Remembers what the user called a food. Saying it again changes nothing.
INSERT INTO food_aliases (alias, food_id, lmv_number) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: GetLMVFood :one
SELECT * FROM lmv.foods WHERE number = $1;

-- name: ListFoods :many
SELECT * FROM foods ORDER BY lower(name), lower(brand);

-- name: GetFood :one
SELECT * FROM foods WHERE id = $1;

-- name: CreateFood :one
INSERT INTO foods (name, brand, gtin, kcal, protein, carbs, fat, fiber, sugars, saturated_fat, salt, notes,
                   search_terms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: UpdateFood :one
UPDATE foods
SET name = $2, brand = $3, gtin = $4, kcal = $5, protein = $6, carbs = $7, fat = $8, fiber = $9,
    sugars = $10, saturated_fat = $11, salt = $12, notes = $13, search_terms = $14, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteFood :execrows
DELETE FROM foods WHERE id = $1;

-- name: PortionsOf :many
-- The portions of the foods named, by either kind of reference.
SELECT * FROM food_portions
WHERE food_id = ANY(sqlc.arg(food_ids)::uuid[]) OR lmv_number = ANY(sqlc.arg(lmv_numbers)::int[])
ORDER BY grams, name;

-- name: SavePortion :one
-- A portion is identified by its food and name, so saving one again changes
-- its weight.
INSERT INTO food_portions (food_id, lmv_number, name, grams)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, coalesce(food_id::text, lmv_number::text), lower(name))
DO UPDATE SET grams = excluded.grams, name = excluded.name
RETURNING *;

-- name: DeletePortion :execrows
DELETE FROM food_portions WHERE id = $1;
