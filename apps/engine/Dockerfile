# syntax=docker/dockerfile:1

# Build stage: compile a small static binary.
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pockdev ./cmd/pockdev

# Run stage: tiny image that idles well under 30MB RAM.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 pockdev
WORKDIR /app
COPY --from=build /out/pockdev /app/pockdev
USER pockdev
EXPOSE 8080
ENTRYPOINT ["/app/pockdev"]
