#!/usr/bin/env bash
source "$(dirname "$0")/common.sh"
"$FORUM_ROOT/scripts/db.sh" up
if [[ "$(podman exec "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='forum_e2e_test'")" != 1 ]]; then
 podman exec "$POSTGRES_CONTAINER" createdb -U "$POSTGRES_USER" forum_e2e_test
fi
export DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@127.0.0.1:${POSTGRES_PORT}/forum_e2e_test?sslmode=disable"
export APP_ORIGIN=http://localhost:18080 API_ADDR=127.0.0.1:18080 STATIC_DIR="$FORUM_ROOT/frontend/dist" COOKIE_SECURE=false E2E_BASE_URL=http://localhost:18080
export E2E_ADMIN_LOGIN="test_$(date +%s)" E2E_ADMIN_PASSWORD="$(openssl rand -hex 16)"
export STAFF_LOGIN="$E2E_ADMIN_LOGIN" STAFF_PASSWORD="$E2E_ADMIN_PASSWORD" STAFF_ROLE=admin
cd "$FORUM_ROOT"
(cd frontend && npm run build)
(cd backend && go build -o bin/forum ./cmd/api)
./backend/bin/forum migrate
./backend/bin/forum seed
./backend/bin/forum create-staff
mkdir -p .run
./backend/bin/forum > .run/e2e-server.log 2>&1 & e2e_pid=$!
trap 'kill "$e2e_pid" 2>/dev/null || true' EXIT INT TERM
for attempt in {1..30}; do
 if ! kill -0 "$e2e_pid" 2>/dev/null; then cat .run/e2e-server.log;exit 1;fi
 if curl -fsS "$E2E_BASE_URL/health" >/dev/null 2>&1;then break;fi
 sleep 1
done
cd frontend
npm test
