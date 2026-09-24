-- The initial schema.
--
-- Two schemas. auth holds identities and credentials and is read before anyone
-- is known, so it has no row level security. public holds a user's training
-- data, and every table in it is owned by a user through user_id, defaults the
-- column from the connection and refuses any row that belongs to somebody else.

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- The user a connection acts for. The pool sets app.user_id when a connection
-- is checked out and clears it on release. An unset value is NULL, and a policy
-- comparing against NULL matches nothing, so a forgotten context fails closed.
CREATE FUNCTION current_user_id() RETURNS uuid
  LANGUAGE sql STABLE
  AS $$ SELECT nullif(current_setting('app.user_id', true), '')::uuid $$;

------------------------------------------------------------------------------
-- auth
------------------------------------------------------------------------------

CREATE SCHEMA auth;

CREATE TABLE auth.users (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  username citext NOT NULL UNIQUE
    CHECK (length(username) BETWEEN 3 AND 32 AND username ~ '^[A-Za-z0-9_.-]+$'),
  -- The WebAuthn user handle. Random rather than the id, so the authenticator
  -- never learns a value the server uses anywhere else.
  webauthn_handle bytea NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.passkeys (
  credential_id bytea PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES auth.users ON DELETE CASCADE,
  public_key bytea NOT NULL,
  attestation_type text NOT NULL DEFAULT '',
  aaguid bytea,
  sign_count bigint NOT NULL DEFAULT 0,
  transports text[] NOT NULL DEFAULT '{}',
  backup_eligible boolean NOT NULL DEFAULT false,
  backup_state boolean NOT NULL DEFAULT false,
  name text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz
);
CREATE INDEX passkeys_user ON auth.passkeys (user_id);

-- A WebAuthn ceremony between its begin and finish calls. The challenge lives
-- here rather than in a cookie, so finishing twice is impossible.
CREATE TABLE auth.webauthn_ceremonies (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  kind text NOT NULL CHECK (kind IN ('signup', 'login', 'add_passkey', 'recover')),
  user_id uuid REFERENCES auth.users ON DELETE CASCADE,
  -- The username a signup will claim, reserved only when it finishes.
  username citext,
  webauthn_handle bytea,
  session_data jsonb NOT NULL,
  expires_at timestamptz NOT NULL
);
CREATE INDEX webauthn_ceremonies_expiry ON auth.webauthn_ceremonies (expires_at);

CREATE TABLE auth.recovery_codes (
  user_id uuid NOT NULL REFERENCES auth.users ON DELETE CASCADE,
  code_hash bytea NOT NULL,
  used_at timestamptz,
  PRIMARY KEY (user_id, code_hash)
);

CREATE TABLE auth.web_sessions (
  token_hash bytea PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES auth.users ON DELETE CASCADE,
  user_agent text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  -- Sliding expiry, extended on use but never past absolute_expires_at.
  expires_at timestamptz NOT NULL,
  absolute_expires_at timestamptz NOT NULL
);
CREATE INDEX web_sessions_user ON auth.web_sessions (user_id);
CREATE INDEX web_sessions_expiry ON auth.web_sessions (expires_at);

-- Proof of work salts already redeemed, kept until the challenge would have
-- expired anyway.
CREATE TABLE auth.pow_spent (
  salt_hash bytea PRIMARY KEY,
  expires_at timestamptz NOT NULL
);

------------------------------------------------------------------------------
-- public: training data
------------------------------------------------------------------------------

-- Every unique constraint starts with user_id. Postgres checks uniqueness
-- before foreign keys, so a constraint without it would answer "duplicate"
-- to somebody probing another user's ids, which confirms the row exists.

CREATE TABLE exercises (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  name text NOT NULL CHECK (btrim(name) <> ''),
  muscle_group text NOT NULL DEFAULT '',
  type text NOT NULL DEFAULT 'strength' CHECK (type IN ('strength', 'cardio')),
  display_type text NOT NULL DEFAULT 'reps' CHECK (display_type IN ('reps', 'time')),
  single_arm boolean NOT NULL DEFAULT false,
  archived boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, user_id)
);
CREATE UNIQUE INDEX exercises_user_name ON exercises (user_id, lower(name));
CREATE INDEX exercises_name_trgm ON exercises USING gin (name gin_trgm_ops);

CREATE TABLE plans (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  name text NOT NULL CHECK (btrim(name) <> ''),
  goal text NOT NULL DEFAULT '',
  notes text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'archived')),
  start_date date,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, user_id)
);
CREATE INDEX plans_user_status ON plans (user_id, status);

-- One entry in a plan's queue. The user picks sessions in position order, or
-- skips them. Whether a session was done is answered by the workout linked to
-- it through workouts.planned_session_id.
CREATE TABLE planned_sessions (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  plan_id uuid NOT NULL,
  position int NOT NULL CHECK (position >= 0),
  label text NOT NULL DEFAULT '',
  week int CHECK (week > 0),
  day text CHECK (length(day) <= 8),
  intensity text CHECK (intensity IN ('heavy', 'light')),
  notes text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'in_progress', 'completed', 'skipped')),
  skip_reason text NOT NULL DEFAULT '',
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, user_id),
  -- Deferred so a reorder can move every position in one transaction.
  CONSTRAINT planned_sessions_position UNIQUE (user_id, plan_id, position) DEFERRABLE INITIALLY DEFERRED,
  FOREIGN KEY (plan_id, user_id) REFERENCES plans (id, user_id) ON DELETE CASCADE
);

CREATE TABLE planned_exercises (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  session_id uuid NOT NULL,
  position int NOT NULL CHECK (position >= 0),
  exercise_id uuid NOT NULL,
  -- Cues and conditional rules, e.g. "if week 6 moved fast, attempt 4 = 182.5".
  notes text NOT NULL DEFAULT '',
  -- Share of the listed weight the non dominant arm works with.
  off_arm_percent numeric(5, 2) CHECK (off_arm_percent > 0 AND off_arm_percent <= 100),
  UNIQUE (id, user_id),
  UNIQUE (user_id, session_id, position),
  FOREIGN KEY (session_id, user_id) REFERENCES planned_sessions (id, user_id) ON DELETE CASCADE,
  FOREIGN KEY (exercise_id, user_id) REFERENCES exercises (id, user_id)
);

-- A group of sets with one prescription, e.g. "Back-off, 3 × 5 @ 130, RPE 7".
CREATE TABLE planned_targets (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  planned_exercise_id uuid NOT NULL,
  position int NOT NULL CHECK (position >= 0),
  set_type text NOT NULL DEFAULT '',
  sets int NOT NULL DEFAULT 1 CHECK (sets BETWEEN 1 AND 50),
  reps int CHECK (reps >= 0),
  weight numeric(7, 2) CHECK (weight >= 0),
  time text NOT NULL DEFAULT '',
  rpe_min numeric(3, 1) CHECK (rpe_min BETWEEN 0 AND 10),
  rpe_max numeric(3, 1) CHECK (rpe_max BETWEEN 0 AND 10),
  notes text NOT NULL DEFAULT '',
  UNIQUE (id, user_id),
  UNIQUE (user_id, planned_exercise_id, position),
  CHECK (rpe_min IS NULL OR rpe_max IS NULL OR rpe_min <= rpe_max),
  FOREIGN KEY (planned_exercise_id, user_id) REFERENCES planned_exercises (id, user_id) ON DELETE CASCADE
);

CREATE TABLE workouts (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  name text NOT NULL DEFAULT '',
  started timestamptz NOT NULL,
  ended timestamptz,
  notes text NOT NULL DEFAULT '',
  planned_session_id uuid,
  -- Bumped on every write, so a stale editor cannot overwrite a newer version.
  revision int NOT NULL DEFAULT 1,
  -- The IndexedDB id of an imported workout, which makes an import repeatable.
  legacy_id bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, user_id),
  UNIQUE (user_id, legacy_id),
  -- One workout per planned session. Scoped to the user, so a unique violation
  -- can never confirm that somebody else's session exists.
  UNIQUE (user_id, planned_session_id),
  FOREIGN KEY (planned_session_id, user_id) REFERENCES planned_sessions (id, user_id)
    ON DELETE SET NULL (planned_session_id)
);
CREATE INDEX workouts_user_started ON workouts (user_id, started DESC);
CREATE INDEX workouts_user_name_started ON workouts (user_id, name, started DESC);

-- An exercise as performed in a workout. The name and attributes are a
-- snapshot, so history keeps reading the way it was logged.
CREATE TABLE workout_exercises (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  workout_id uuid NOT NULL,
  position int NOT NULL CHECK (position >= 0),
  exercise_id uuid,
  name text NOT NULL,
  muscle_group text NOT NULL DEFAULT '',
  type text NOT NULL DEFAULT 'strength' CHECK (type IN ('strength', 'cardio')),
  display_type text NOT NULL DEFAULT 'reps' CHECK (display_type IN ('reps', 'time')),
  single_arm boolean NOT NULL DEFAULT false,
  intensity text CHECK (intensity IN ('heavy', 'light')),
  planned_exercise_id uuid,
  notes text NOT NULL DEFAULT '',
  UNIQUE (id, user_id),
  UNIQUE (user_id, workout_id, position),
  FOREIGN KEY (workout_id, user_id) REFERENCES workouts (id, user_id) ON DELETE CASCADE,
  FOREIGN KEY (exercise_id, user_id) REFERENCES exercises (id, user_id) ON DELETE SET NULL (exercise_id),
  FOREIGN KEY (planned_exercise_id, user_id) REFERENCES planned_exercises (id, user_id)
    ON DELETE SET NULL (planned_exercise_id)
);
CREATE INDEX workout_exercises_user_name ON workout_exercises (user_id, lower(name));
CREATE INDEX workout_exercises_planned ON workout_exercises (planned_exercise_id);

CREATE TABLE workout_sets (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  workout_exercise_id uuid NOT NULL,
  position int NOT NULL CHECK (position >= 0),
  type text NOT NULL DEFAULT 'regular' CHECK (type IN ('regular', 'warmup')),
  weight numeric(7, 2),
  distance numeric(9, 3),
  reps int CHECK (reps >= 0),
  time text NOT NULL DEFAULT '',
  rpe numeric(3, 1) CHECK (rpe BETWEEN 0 AND 10),
  arm text NOT NULL DEFAULT '' CHECK (arm IN ('', 'left', 'right', 'both')),
  notes text NOT NULL DEFAULT '',
  -- The planned target this set was prescribed by, and which of its sets it is.
  target_id uuid,
  target_seq int CHECK (target_seq > 0),
  UNIQUE (id, user_id),
  UNIQUE (user_id, workout_exercise_id, position),
  FOREIGN KEY (workout_exercise_id, user_id) REFERENCES workout_exercises (id, user_id) ON DELETE CASCADE,
  FOREIGN KEY (target_id, user_id) REFERENCES planned_targets (id, user_id) ON DELETE SET NULL (target_id)
);
CREATE INDEX workout_sets_target ON workout_sets (target_id);

-- Templates are only ever read and written whole, so they stay a document.
CREATE TABLE templates (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  name text NOT NULL DEFAULT '',
  doc jsonb NOT NULL,
  legacy_id bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, user_id),
  UNIQUE (user_id, legacy_id)
);

CREATE TABLE settings (
  user_id uuid NOT NULL DEFAULT current_user_id() REFERENCES auth.users ON DELETE CASCADE,
  key text NOT NULL CHECK (key ~ '^[A-Za-z0-9_.-]{1,64}$'),
  value jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, key)
);

------------------------------------------------------------------------------
-- Row level security
------------------------------------------------------------------------------

DO $$
DECLARE
  t text;
BEGIN
  FOREACH t IN ARRAY ARRAY[
    'exercises', 'plans', 'planned_sessions', 'planned_exercises', 'planned_targets',
    'workouts', 'workout_exercises', 'workout_sets', 'templates', 'settings'
  ] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format(
      'CREATE POLICY owner_only ON %I USING (user_id = current_user_id()) WITH CHECK (user_id = current_user_id())',
      t);
  END LOOP;
END
$$;

------------------------------------------------------------------------------
-- Grants
------------------------------------------------------------------------------

-- Grants are part of the schema rather than the deployment. The compose init
-- script sets default privileges for a developer database, but a database the
-- test provisioner clones from a template does not inherit them, and the suite
-- must run with exactly the privileges production has.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'plonkout_app') THEN
    RETURN;
  END IF;
  GRANT USAGE ON SCHEMA public, auth TO plonkout_app;
  GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public, auth TO plonkout_app;
  GRANT EXECUTE ON FUNCTION current_user_id() TO plonkout_app;
  ALTER DEFAULT PRIVILEGES IN SCHEMA public, auth
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO plonkout_app;
END
$$;
