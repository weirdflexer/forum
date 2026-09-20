#!/usr/bin/env bash
source "$(dirname "$0")/common.sh"
case "${1:-up}" in
 up)
  if ! podman info >/dev/null 2>&1; then
   if [[ "$(uname)" == Darwin ]]; then
    if [[ -z "$(podman machine list --format '{{.Name}}')" ]]; then podman machine init; fi
    podman machine start
   else echo 'Запустите Podman и повторите команду.' >&2; exit 1; fi
  fi
  if podman container exists "$POSTGRES_CONTAINER"; then podman start "$POSTGRES_CONTAINER" >/dev/null
  else
   if ! podman volume exists forum-postgres-data; then
    podman volume create forum-postgres-data >/dev/null
   fi
   podman run -d --name "$POSTGRES_CONTAINER" --label app=anonymous-forum --restart unless-stopped \
    -p "127.0.0.1:${POSTGRES_PORT}:5432" \
    -e POSTGRES_USER -e POSTGRES_PASSWORD -e POSTGRES_DB \
    -v forum-postgres-data:/var/lib/postgresql/data \
    --health-cmd 'pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"' --health-interval 10s \
    docker.io/library/postgres:17-alpine >/dev/null
  fi
  for attempt in {1..30}; do
   if podman exec "$POSTGRES_CONTAINER" pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then echo "PostgreSQL готова: 127.0.0.1:$POSTGRES_PORT";exit 0;fi
   sleep 1
  done
  echo 'PostgreSQL не запустилась. Проверьте podman logs forum-postgres.' >&2; exit 1;;
 stop) podman stop "$POSTGRES_CONTAINER";;
 logs) podman logs --tail 100 "$POSTGRES_CONTAINER";;
 status) podman ps -a --filter "name=$POSTGRES_CONTAINER";;
 *) echo 'Использование: db.sh up|stop|logs|status' >&2; exit 1;;
esac
