.PHONY: dev run build migrate docker-up docker-down

dev:
	air

run:
	go run cmd/server/main.go

build:
	go build -o bin/autobidd cmd/server/main.go

migrate:
	go run cmd/server/main.go -migrate

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
