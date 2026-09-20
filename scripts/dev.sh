#!/usr/bin/env bash
source "$(dirname "$0")/common.sh"
"$FORUM_ROOT/scripts/db.sh" up
"$FORUM_ROOT/scripts/backend.sh" migrate
mkdir -p "$FORUM_ROOT/.run"
(cd "$FORUM_ROOT/backend" && go build -o "$FORUM_ROOT/.run/forum-dev" ./cmd/api)
"$FORUM_ROOT/.run/forum-dev" & api_pid=$!
(cd "$FORUM_ROOT/frontend" && exec node node_modules/vite/bin/vite.js --host 127.0.0.1) & front_pid=$!
trap 'kill "$api_pid" "$front_pid" 2>/dev/null || true' EXIT INT TERM
wait "$api_pid" "$front_pid"
