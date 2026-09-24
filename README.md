# Plonkout

A workout logger for the phone, with training plans an AI assistant can write and review. Built with a Vue 3 PWA and a Go server on Postgres.

## Features

- **Workout logging**: sets with weight, reps, time, RPE and arm, warmups, notes and auto save
- **Training plans**: an ordered queue of sessions with targets. Pick the next one, skip it, and see planned against done
- **Assistant ready**: connect Claude or any MCP client to design plans from your history and adjust them as you go
- **Armwrestling first**: single arm exercises, a dominant arm, off arm percentages and per arm history
- **Passkeys**: sign in with your face or fingerprint, no passwords. Recovery codes for when every device is lost
- **Statistics**: records, estimated 1RM and charts
- **English and Swedish**
- **Installable PWA** with a neobrutalism design

## How it fits together

```
web/      Vue 3 + TypeScript PWA, served by the server in production
server/   Go: huma API, OAuth 2.1 authorization server, MCP endpoint
          Postgres with row level security, one policy per user table
```

- The API is described by `server/api/openapi.json`, generated from the Go code. The web client in `web/src/api/gen` is generated from it.
- Every user table carries `user_id`, filled from the connection. The server connects as a role that cannot bypass row level security, so a query can only ever see the signed in user's rows.
- An MCP tool call is a request through the same API with the caller's access token, so an assistant reaches exactly what its user could.

## Development

You need Go 1.27, Node 25 with pnpm, Docker and [sqlc](https://sqlc.dev).

```bash
make up                                  # Postgres on 127.0.0.1:5449
cp server/config.example.toml server/config.toml
make run                                 # the server on :8090, migrating first
make web                                 # Vite on :5173, proxying the API to the server
```

Open http://localhost:5173. Passkeys work on `localhost` without https.

| Command         | What it does                                                 |
| --------------- | ------------------------------------------------------------ |
| `make generate` | sqlc, the OpenAPI document and the web client                |
| `make test`     | Go tests against real Postgres databases, then the web tests |
| `make lint`     | go vet, gofmt and eslint                                     |
| `make check`    | everything CI runs, and fails when generated code is stale   |
| `make docker`   | builds the image                                             |

Go tests create a fresh database per test from a migrated template (pgtestdb), so they need `make up` first.

## Deploying

CI publishes `ghcr.io/zanmato/plonkout` for every push to main and every `v*` tag. The image migrates on start and serves the app, the API, OAuth and MCP on one port.

```bash
docker run -p 8080:8080 \
  -e PLONKOUT_SERVER_BASE_URL=https://plonkout.example \
  -e PLONKOUT_DATABASE_APP=postgres://plonkout_app:...@db:5432/plonkout \
  -e PLONKOUT_DATABASE_MIGRATE=postgres://plonkout_migrate:...@db:5432/plonkout \
  -e PLONKOUT_AUTH_SECRET_KEY=$(openssl rand -hex 32) \
  ghcr.io/zanmato/plonkout
```

Create the two roles with `docker/postgres/initdb/10-roles.sh`. The app role must not own the schema or bypass row level security. The server has to be reached over https at the base URL, since passkeys and the session cookie depend on it.

## Connecting Claude

In Claude, add a custom connector with the URL `https://plonkout.example/mcp`. For Claude Code:

```bash
claude mcp add --transport http plonkout https://plonkout.example/mcp
```

Sign in with your passkey when asked and approve the connection. Then ask it to design a plan, or use the `design_plan` and `weekly_review` prompts. Connected apps are listed, and can be disconnected, under Settings.

## Moving from the local only version

Export your data in the old app (Settings, Export data), sign up here and import the file under Settings. Importing the same file twice skips what is already there.

## License

This project is open source and available under the [MIT License](LICENSE).
