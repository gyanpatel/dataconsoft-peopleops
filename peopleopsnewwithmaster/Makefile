.PHONY: run test build web-install web-build docker-up
run:
	go run ./cmd/server

test:
	go test ./...

build:
	go build -trimpath -o bin/peopleops ./cmd/server

web-install:
	cd web && npm install

web-build:
	cd web && npm run build

docker-up:
	docker compose up --build -d
