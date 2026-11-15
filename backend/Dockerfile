# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /src

# git needed by `go install` for module resolution.
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the server, the seeder, and the golang-migrate CLI.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/seed   ./cmd/seed
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1

# ---- runtime stage ----
FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=build /out/server ./server
COPY --from=build /out/seed   ./seed
COPY --from=build /go/bin/migrate /usr/local/bin/migrate
COPY db/migrations ./migrations
COPY entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]

# ---- test stage ----
# Used by the compose `test` service (profile: test). Has the Go toolchain
# so tests can be compiled and run inside the container.
FROM golang:1.25-alpine AS test
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
# Apply migrations, then run the tests.
CMD migrate -path /src/db/migrations \
    -database "postgres://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable" up \
    && go test -v ./...
