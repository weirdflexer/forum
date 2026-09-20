#!/usr/bin/env bash
source "$(dirname "$0")/common.sh"
"$FORUM_ROOT/scripts/db.sh" up
if [[ "$(podman exec "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='forum_test'")" != 1 ]]; then
 podman exec "$POSTGRES_CONTAINER" createdb -U "$POSTGRES_USER" forum_test
fi
export TEST_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@127.0.0.1:${POSTGRES_PORT}/forum_test?sslmode=disable"
cd "$FORUM_ROOT/backend"
go test -race -count=1 -v ./...
