#!/usr/bin/env bash
source "$(dirname "$0")/common.sh"
cd "$FORUM_ROOT"
mkdir -p .run
if [[ -f .run/server.pid ]] && kill -0 "$(cat .run/server.pid)" 2>/dev/null; then echo 'Форум уже запущен.'; exit 0; fi
./scripts/db.sh up
./scripts/backend.sh migrate
make build
# Single-origin production build served by Go on localhost. Database remains in Podman.
APP_ORIGIN=http://localhost:8080 STATIC_DIR="$FORUM_ROOT/frontend/dist" nohup ./backend/bin/forum >.run/server.log 2>&1 </dev/null &
pid=$!
echo "$pid" >.run/server.pid
for attempt in {1..20}; do
 if ! kill -0 "$pid" 2>/dev/null; then cat .run/server.log; exit 1;fi
 if curl -fsS http://127.0.0.1:8080/health >/dev/null 2>&1; then echo 'Форум открыт: http://localhost:8080';exit 0;fi
 sleep 1
done
echo 'Не удалось дождаться запуска. См. .run/server.log' >&2; exit 1
