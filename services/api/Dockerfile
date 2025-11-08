# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/service ./services/api/cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/service /app/service
WORKDIR /app
ENTRYPOINT ["/app/service"]
