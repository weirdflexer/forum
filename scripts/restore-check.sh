#!/usr/bin/env bash
source "$(dirname "$0")/common.sh"
if [[ $# -ne 1 || ! -f "$1" ]]; then echo 'Использование: ./scripts/restore-check.sh backups/forum-....dump' >&2;exit 1;fi
# Restore into a NEW database; never overwrite live data.
restore_db="forum_restore_$(date -u +%Y%m%d%H%M%S)"
podman exec "$POSTGRES_CONTAINER" createdb -U "$POSTGRES_USER" "$restore_db"
podman exec -i "$POSTGRES_CONTAINER" pg_restore -U "$POSTGRES_USER" -d "$restore_db" --exit-on-error --no-owner < "$1"
podman exec "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$restore_db" -c 'SELECT (SELECT count(*) FROM topics) AS topics, (SELECT count(*) FROM posts) AS posts, (SELECT count(*) FROM sections) AS sections;'
echo "Проверка восстановления выполнена в отдельной БД $restore_db. Рабочая БД не изменена."
