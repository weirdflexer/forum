#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -f .run/server.pid ]]; then
 pid="$(cat .run/server.pid)"
 # Do not signal an unrelated process if the PID has been recycled.
 if ps -p "$pid" -o command= | grep -Fq './backend/bin/forum'; then kill "$pid"; fi
 rm .run/server.pid
fi
echo 'Сервер остановлен. База остаётся запущенной; make db-stop остановит её без удаления данных.'
