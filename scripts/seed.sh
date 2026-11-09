#!/usr/bin/env sh
set -e

ACTION="${1:-up}"
case "$ACTION" in
    down)
        exec go run ./cmd/seed -down
        ;;
    reset)
        exec go run ./cmd/seed -reset
        ;;
    up|*)
        exec go run ./cmd/seed
        ;;
esac
