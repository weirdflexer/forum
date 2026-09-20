#!/usr/bin/env bash
source "$(dirname "$0")/common.sh"
cd "$FORUM_ROOT/backend"
exec go run ./cmd/api "${1:-serve}"
