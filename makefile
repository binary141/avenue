target: up-d-build

up-d:
	docker compose up -d

up-d-build:
	docker compose up -d --build

down:
	docker compose down

run:
	go run ./...

lint:
	golangci-lint run

build-api:
	go build -o api ./

build-api-prod:
	go build -tags prod -o api ./

build-ui:
	npm run build
