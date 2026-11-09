#!/usr/bin/env sh
set -e

exec go test -bench=Benchmark -benchmem -run=none -benchtime=2s ./pkg/logger
