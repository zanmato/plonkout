-- The OAuth 2.1 authorization server MCP clients sign in through.
--
-- Tokens are opaque and only their hashes are stored. Looking one up is a
-- single indexed read, and revoking a grant takes effect on the next request.

-- A client, registered dynamically (RFC 7591) or described by a metadata
-- document at an https client_id (Client ID Metadata Documents), which is
-- fetched and cached here.
CREATE TABLE auth.oauth_clients (
  client_id text PRIMARY KEY,
  kind text NOT NULL CHECK (kind IN ('dcr', 'cimd')),
  client_name text NOT NULL DEFAULT '',
  redirect_uris text[] NOT NULL,
  metadata jsonb NOT NULL DEFAULT '{}',
  -- When a metadata document must be fetched again.
  cache_until timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz
);

-- An authorization code, single use and short lived.
CREATE TABLE auth.oauth_codes (
  code_hash bytea PRIMARY KEY,
  client_id text NOT NULL REFERENCES auth.oauth_clients ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES auth.users ON DELETE CASCADE,
  redirect_uri text NOT NULL,
  code_challenge text NOT NULL,
  scope text NOT NULL,
  resource text NOT NULL,
  expires_at timestamptz NOT NULL
);

-- What a user approved for a client: shown as a connected app, revocable.
-- The refresh token rotates on every use. The previous one is remembered, so
-- presenting it again is recognized as a stolen token and ends the grant.
CREATE TABLE auth.oauth_grants (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  client_id text NOT NULL REFERENCES auth.oauth_clients ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES auth.users ON DELETE CASCADE,
  scope text NOT NULL,
  resource text NOT NULL,
  refresh_hash bytea UNIQUE,
  previous_refresh_hash bytea UNIQUE,
  refresh_expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz,
  revoked_at timestamptz
);
CREATE INDEX oauth_grants_user ON auth.oauth_grants (user_id);

CREATE TABLE auth.oauth_access_tokens (
  token_hash bytea PRIMARY KEY,
  grant_id uuid NOT NULL REFERENCES auth.oauth_grants ON DELETE CASCADE,
  expires_at timestamptz NOT NULL
);
CREATE INDEX oauth_access_tokens_expiry ON auth.oauth_access_tokens (expires_at);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'plonkout_app') THEN
    GRANT SELECT, INSERT, UPDATE, DELETE ON
      auth.oauth_clients, auth.oauth_codes, auth.oauth_grants, auth.oauth_access_tokens
      TO plonkout_app;
  END IF;
END
$$;
