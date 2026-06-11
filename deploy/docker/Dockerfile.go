# syntax=docker/dockerfile:1
# Parameterized multi-stage build for every Go service (D-010/D-029):
#   docker build -f deploy/docker/Dockerfile.go --build-arg SERVICE=services/ingestion -t membrane/ingestion:dev .
# Build context = repo root (the module `replace` directives need ../../pkg and ../../proto).
# Output: a static binary on distroless/static — no shell, no libc coupling, runs as nonroot.

ARG GO_VERSION=1.25

FROM golang:${GO_VERSION} AS build
ARG SERVICE
WORKDIR /src

# Warm the module cache first so code edits don't re-download dependencies;
# the BuildKit cache mounts are shared across ALL service builds.
COPY pkg/go.mod pkg/go.sum* ./pkg/
COPY proto/go.mod proto/go.sum* ./proto/
COPY ${SERVICE}/go.mod ${SERVICE}/go.sum* ./${SERVICE}/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd ${SERVICE} && GOWORK=off go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd ${SERVICE} \
 && GOWORK=off CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/*

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
