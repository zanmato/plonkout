-- Calorie tracking: a food diary with goals, the user's own foods, and the
-- Swedish Food Agency's food database to look foods up in.

------------------------------------------------------------------------------
-- lmv: Livsmedelsverkets livsmedelsdatabas (CC BY 4.0)
------------------------------------------------------------------------------

-- Reference data shared by every user, so like auth it has no row level
-- security. The server synchronizes it from the published file.
CREATE SCHEMA lmv;

-- One row per release imported. The file is byte for byte the same until
-- Livsmedelsverket publishes a new version, so its hash says whether a
-- download holds anything new.
CREATE TABLE lmv.releases (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  version text NOT NULL,
  sha256 bytea NOT NULL UNIQUE,
  foods int NOT NULL,
  imported_at timestamptz NOT NULL DEFAULT now()
);

-- A food per 100 g edible part. The main nutrients are columns, every value
-- the release has is kept in nutrients under its Swedish column name.
CREATE TABLE lmv.foods (
  number int PRIMARY KEY,
  name text NOT NULL,
  food_group text NOT NULL DEFAULT '',
  kcal numeric(7, 2) NOT NULL,
  protein numeric(6, 2) NOT NULL,
  carbs numeric(6, 2) NOT NULL,
  fat numeric(6, 2) NOT NULL,
  fiber numeric(6, 2),
  sugars numeric(6, 2),
  saturated_fat numeric(6, 2),
  salt numeric(6, 2),
  nutrients jsonb NOT NULL DEFAULT '{}',
  -- A food a later release no longer has. Kept, since portions point at it.
  retired boolean NOT NULL DEFAULT false,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX lmv_foods_name_trgm ON lmv.foods USING gin (name gin_trgm_ops);

------------------------------------------------------------------------------
-- public: the user's foods and diary
------------------------------------------------------------------------------

-- A food the user added, typically a product from its label.
CREATE TABLE foods (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  name text NOT NULL CHECK (btrim(name) <> ''),
  brand text NOT NULL DEFAULT '',
  gtin text CHECK (gtin ~ '^[0-9]{8,14}$'),
  kcal numeric(7, 2) NOT NULL CHECK (kcal >= 0),
  protein numeric(6, 2) NOT NULL DEFAULT 0 CHECK (protein >= 0),
  carbs numeric(6, 2) NOT NULL DEFAULT 0 CHECK (carbs >= 0),
  fat numeric(6, 2) NOT NULL DEFAULT 0 CHECK (fat >= 0),
  fiber numeric(6, 2) CHECK (fiber >= 0),
  sugars numeric(6, 2) CHECK (sugars >= 0),
  saturated_fat numeric(6, 2) CHECK (saturated_fat >= 0),
  salt numeric(6, 2) CHECK (salt >= 0),
  notes text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, user_id)
);
CREATE UNIQUE INDEX foods_user_name ON foods (user_id, lower(name), lower(brand));
CREATE UNIQUE INDEX foods_user_gtin ON foods (user_id, gtin);
CREATE INDEX foods_name_trgm ON foods USING gin (name gin_trgm_ops);

-- A household measure of a food, e.g. a slice is 10 g or a tablespoon 15 g.
-- Either of the user's own foods or of a Livsmedelsverket food.
CREATE TABLE food_portions (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  food_id uuid,
  lmv_number int REFERENCES lmv.foods,
  name text NOT NULL CHECK (btrim(name) <> '' AND length(name) <= 40),
  grams numeric(7, 2) NOT NULL CHECK (grams > 0),
  UNIQUE (id, user_id),
  CHECK (num_nonnulls(food_id, lmv_number) = 1),
  FOREIGN KEY (food_id, user_id) REFERENCES foods (id, user_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX food_portions_user_name
  ON food_portions (user_id, coalesce(food_id::text, lmv_number::text), lower(name));

-- Something eaten. The name and nutrients per 100 g are a snapshot, so a food
-- edited later or revised by a new release leaves the day as it was logged.
CREATE TABLE food_entries (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  -- The user's own calendar day.
  day date NOT NULL,
  meal text NOT NULL CHECK (meal IN ('breakfast', 'lunch', 'dinner', 'snack')),
  food_id uuid,
  lmv_number int REFERENCES lmv.foods,
  name text NOT NULL CHECK (btrim(name) <> ''),
  grams numeric(7, 1) NOT NULL CHECK (grams > 0 AND grams <= 5000),
  -- How the amount was put, e.g. "2 tbsp", next to the grams it came to.
  amount text NOT NULL DEFAULT '',
  kcal numeric(7, 2) NOT NULL CHECK (kcal >= 0),
  protein numeric(6, 2) NOT NULL CHECK (protein >= 0),
  carbs numeric(6, 2) NOT NULL CHECK (carbs >= 0),
  fat numeric(6, 2) NOT NULL CHECK (fat >= 0),
  fiber numeric(6, 2),
  sugars numeric(6, 2),
  saturated_fat numeric(6, 2),
  salt numeric(6, 2),
  logged_by text NOT NULL DEFAULT 'app' CHECK (logged_by IN ('app', 'assistant')),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, user_id),
  FOREIGN KEY (food_id, user_id) REFERENCES foods (id, user_id) ON DELETE SET NULL (food_id)
);
CREATE INDEX food_entries_user_day ON food_entries (user_id, day);

-- Energy burned by exercise, entered by hand or estimated by an assistant.
CREATE TABLE activities (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  day date NOT NULL,
  label text NOT NULL CHECK (btrim(label) <> ''),
  kcal int NOT NULL CHECK (kcal > 0 AND kcal <= 10000),
  workout_id uuid,
  logged_by text NOT NULL DEFAULT 'app' CHECK (logged_by IN ('app', 'assistant')),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, user_id),
  FOREIGN KEY (workout_id, user_id) REFERENCES workouts (id, user_id) ON DELETE SET NULL (workout_id)
);
CREATE INDEX activities_user_day ON activities (user_id, day);

-- The daily budget and what it is for. One per user.
CREATE TABLE nutrition_goals (
  user_id uuid PRIMARY KEY DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  direction text NOT NULL DEFAULT 'maintain' CHECK (direction IN ('lose', 'maintain', 'gain')),
  kcal int NOT NULL CHECK (kcal BETWEEN 800 AND 10000),
  protein int NOT NULL CHECK (protein >= 0),
  carbs int NOT NULL CHECK (carbs >= 0),
  fat int NOT NULL CHECK (fat >= 0),
  -- Whether energy burned by activities is added to the day's budget.
  add_activities boolean NOT NULL DEFAULT true,
  -- In the user's weight unit.
  target_weight numeric(6, 2) CHECK (target_weight > 0),
  weekly_change numeric(4, 2) CHECK (weekly_change BETWEEN -2 AND 2),
  notes text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- Body weight, one per day, in the user's weight unit.
CREATE TABLE body_weights (
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  day date NOT NULL,
  weight numeric(6, 2) NOT NULL CHECK (weight > 0 AND weight < 1000),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, day)
);

DO $$
DECLARE
  t text;
BEGIN
  FOREACH t IN ARRAY ARRAY[
    'foods', 'food_portions', 'food_entries', 'activities', 'nutrition_goals', 'body_weights'
  ] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format(
      'CREATE POLICY owner_only ON %I USING (user_id = current_user_id()) WITH CHECK (user_id = current_user_id())',
      t);
  END LOOP;
END
$$;

-- The public tables are covered by the default privileges of the initial
-- migration. The server imports releases itself, so it writes to lmv too.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'plonkout_app') THEN
    RETURN;
  END IF;
  GRANT USAGE ON SCHEMA lmv TO plonkout_app;
  GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA lmv TO plonkout_app;
  GRANT SELECT, INSERT, UPDATE, DELETE ON foods, food_portions, food_entries, activities,
    nutrition_goals, body_weights TO plonkout_app;
END
$$;
