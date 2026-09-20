#!/usr/bin/env bash
set -euo pipefail
FORUM_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ ! -f "$FORUM_ROOT/.env" ]]; then echo 'Сначала выполните ./scripts/setup.sh' >&2; exit 1; fi
set -a
source "$FORUM_ROOT/.env"
set +a
