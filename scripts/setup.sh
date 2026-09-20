#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
for tool in go node npm podman openssl; do command -v "$tool" >/dev/null || { echo "Не найден $tool" >&2; exit 1; }; done
if [[ ! -f .env ]]; then
 umask 077
 db_secret="$(openssl rand -hex 24)"
 session_secret="$(openssl rand -hex 32)"
 cat > .env <<ENV
POSTGRES_USER=forum
POSTGRES_DB=forum
POSTGRES_PASSWORD=$db_secret
POSTGRES_PORT=55432
POSTGRES_CONTAINER=forum-postgres
DATABASE_URL=postgres://forum:$db_secret@127.0.0.1:55432/forum?sslmode=disable
SESSION_SECRET=$session_secret
APP_ORIGIN=http://localhost:5173
API_ADDR=127.0.0.1:8080
COOKIE_SECURE=false
ENV
 echo '.env создан со случайными секретами.'
fi
(cd backend && go mod download)
(cd frontend && npm ci)
echo 'Готово. Далее: make db-up migrate seed'
