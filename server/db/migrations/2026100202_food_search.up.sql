-- Food search that understands Swedish: stemming through the swedish text
-- search configuration, and compound words split at import time.
--
-- search_terms holds the extra words the server derives from a name, such as
-- "kyckling färs" for kycklingfärs or "korv" for falukorv. They are weighted
-- below the words of the name itself, so a food named for a word ranks above
-- one that merely ends with it.

ALTER TABLE lmv.foods
  ADD COLUMN search_terms text NOT NULL DEFAULT '',
  ADD COLUMN search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('swedish', name), 'A') || setweight(to_tsvector('swedish', search_terms), 'C')
  ) STORED;
CREATE INDEX lmv_foods_search ON lmv.foods USING gin (search);

-- The words a release's compounds are split into, learned from its names.
-- Queries and the user's own foods are split with the same words.
ALTER TABLE lmv.releases ADD COLUMN vocabulary text[] NOT NULL DEFAULT '{}';

ALTER TABLE foods
  ADD COLUMN search_terms text NOT NULL DEFAULT '',
  ADD COLUMN search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('swedish', name || ' ' || brand), 'A') || setweight(to_tsvector('swedish', search_terms), 'C')
  ) STORED;
CREATE INDEX foods_search ON foods USING gin (search);

-- What the user calls a food, e.g. "ölkorv" for a sausage the database names
-- otherwise. Learned when an assistant logs a food it had to search for.
CREATE TABLE food_aliases (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  alias text NOT NULL CHECK (btrim(alias) <> '' AND length(alias) <= 60),
  food_id uuid,
  lmv_number int REFERENCES lmv.foods,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, user_id),
  CHECK (num_nonnulls(food_id, lmv_number) = 1),
  FOREIGN KEY (food_id, user_id) REFERENCES foods (id, user_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX food_aliases_user_alias
  ON food_aliases (user_id, lower(alias), coalesce(food_id::text, lmv_number::text));

ALTER TABLE food_aliases ENABLE ROW LEVEL SECURITY;
CREATE POLICY owner_only ON food_aliases
  USING (user_id = current_user_id()) WITH CHECK (user_id = current_user_id());

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'plonkout_app') THEN
    RETURN;
  END IF;
  GRANT SELECT, INSERT, UPDATE, DELETE ON food_aliases TO plonkout_app;
END
$$;

-- Releases imported before this have no search terms. Forgetting them makes
-- the server import the current release again when it next starts.
DELETE FROM lmv.releases;
