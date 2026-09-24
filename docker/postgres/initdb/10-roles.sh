#!/bin/bash
# Creates the plonkout roles and database.
#
# plonkout_migrate owns the schema and runs migrations.
# plonkout_app is what the server connects as. It must never bypass RLS.
set -euo pipefail

: "${PLONKOUT_MIGRATE_PASSWORD:=plonkout_migrate}"
: "${PLONKOUT_APP_PASSWORD:=plonkout_app}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<SQL
CREATE ROLE plonkout_migrate LOGIN PASSWORD '${PLONKOUT_MIGRATE_PASSWORD}' CREATEDB NOSUPERUSER NOCREATEROLE;
CREATE ROLE plonkout_app LOGIN PASSWORD '${PLONKOUT_APP_PASSWORD}' NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;

CREATE DATABASE plonkout OWNER plonkout_migrate;
SQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname plonkout <<SQL
GRANT CONNECT ON DATABASE plonkout TO plonkout_app;
SQL
