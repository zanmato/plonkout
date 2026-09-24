.PHONY: up down reset-db migrate run web sqlc openapi client generate test test-go test-web lint build-web check docker

up:
	docker compose up -d --wait

down:
	docker compose down

# Drops the volume, so the roles init script runs again.
reset-db:
	docker compose down -v
	docker compose up -d --wait

migrate:
	cd server && go run ./cmd/plonkout -migrate-only

# The server on :8090. Pair it with `make web`, whose dev server proxies to it.
run:
	cd server && go run ./cmd/plonkout -m

web:
	cd web && pnpm dev

sqlc:
	cd server && sqlc generate

openapi:
	cd server && go run ./cmd/plonkout openapi > api/openapi.json

client: openapi
	cd web && pnpm generate

generate: sqlc client

test-go:
	cd server && go test -race ./...

test-web:
	cd web && pnpm type-check && pnpm test --run

test: test-go test-web

lint:
	cd server && go vet ./...
	@cd server && test -z "$$(gofmt -l .)" || { gofmt -l .; echo 'run gofmt -w'; exit 1; }
	cd web && pnpm lint

build-web:
	cd web && pnpm build

# Everything CI runs. Generated code must be committed and current.
check: generate lint test build-web
	git diff --exit-code -- server/api server/internal web/src/api/gen
	@# git diff cannot see a generated file that was never committed.
	@test -z "$$(git status --porcelain -- server/api server/internal web/src/api/gen)" || { git status --short -- server/api server/internal web/src/api/gen; echo "generated files are missing from the commit"; exit 1; }

docker:
	docker build -t plonkout:dev .
