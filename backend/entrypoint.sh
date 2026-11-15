#!/bin/sh
# Backend container entrypoint: runs pending migrations, then starts the server.
set -e

MIGRATIONS_DIR="${MIGRATIONS_DIR:-/app/migrations}"
DATABASE_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT:-5432}/${DB_NAME}?sslmode=disable"

echo "running migrations from ${MIGRATIONS_DIR}"
migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" up

echo "starting server"
exec ./server
