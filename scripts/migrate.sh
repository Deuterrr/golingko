#!/usr/bin/env sh
set -e

ACTION="${1:-up}"
if [ $# -gt 0 ]; then
    shift
fi

exec go run ./cmd/migrate "$ACTION" "$@"
