#!/usr/bin/env sh
set -e

exec go run ./cmd/stress -matrix "$@"
