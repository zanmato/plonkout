# syntax=docker/dockerfile:1

FROM node:25-bookworm-slim AS web
WORKDIR /src/web
# Node 25 no longer ships corepack, so pnpm comes from npm at the pinned version.
RUN npm install -g pnpm@10.11.1
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.27-bookworm AS server
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY server/ ./
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X github.com/zanmato/plonkout/server/internal/server.Version=${VERSION}" \
      -o /out/plonkout ./cmd/plonkout && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/healthchecker ./cmd/healthchecker

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=server /out/plonkout /out/healthchecker /app/
COPY server/db/migrations /app/migrations
COPY --from=web /src/web/dist /app/web
COPY deploy/config.toml /app/config.toml
ENV PLONKOUT_CONFIG=/app/config.toml
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s CMD ["/app/healthchecker"]
# -m migrates before serving. The migrate connection is only used at startup.
ENTRYPOINT ["/app/plonkout", "-m"]
