.PHONY: install up down test test-e2e lint swagger seed

install:
	docker-compose build

up:
	docker-compose up -d

down:
	docker-compose down

test:
	docker build -t notification-hub-test -f Dockerfile.test .
	docker run --rm notification-hub-test

test-e2e:
	docker-compose up -d
	@echo "Waiting for services..."
	@sleep 10
	go test ./test/e2e/... -v -tags=e2e -timeout 120s

lint:
	golangci-lint run

swagger:
	swag init -g cmd/api/main.go -o docs

seed:
	go run seed/seeder.go
