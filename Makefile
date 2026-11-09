.PHONY: up down dev test bench bench-loggers stress stress-matrix research-matrix build clean test-api-bruno test-api-postman migrate-up migrate-down migrate-status migrate-create seed seed-down seed-reset

up:
	docker compose up -d

down:
	docker compose down

dev:
	go run ./cmd/golingko

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down 1

migrate-status:
	go run ./cmd/migrate version

migrate-create:
	go run ./cmd/migrate create $(name)

seed:
	go run ./cmd/seed

seed-down:
	go run ./cmd/seed -down

seed-reset:
	go run ./cmd/seed -reset

test:
	go test -v -race ./...

bench:
	go test -bench Benchmark -benchmem -run none ./test ./pkg/logger

bench-loggers:
	go test -bench=Benchmark -benchmem -run=none -benchtime=2s ./pkg/logger

stress:
	go run ./cmd/stress -workers 200 -duration 5s

stress-matrix:
	go run ./cmd/stress -matrix

research-matrix:
	go run ./cmd/stress -research

build:
	go build -o bin/golingko ./cmd/golingko

clean:
	rm -rf bin/

test-api-bruno:
	npx @usebruno/cli run api/bruno --env Local

test-api-postman:
	npx newman run api/postman/golingko.postman_collection.json -e api/postman/golingko_local.postman_environment.json
