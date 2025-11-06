# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/service ./services/orchestrator/cmd/orchestrator

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=builder /out/service /app/service
WORKDIR /app
ENTRYPOINT ["/app/service"]
