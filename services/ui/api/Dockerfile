# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X goosed/services/ui/api/internal/version.Version=$(git rev-parse --short HEAD 2>/dev/null || echo dev)" -o /out/ui-api ./cmd/ui-api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/ui-api /ui-api
USER nonroot:nonroot
ENTRYPOINT ["/ui-api"]
