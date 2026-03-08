.PHONY: build run test test-e2e test-coverage lint swagger migrate-up migrate-down seed docker-up docker-down clean

build:
	go build -o bin/notification-hub cmd/api/main.go

run:
	go run cmd/api/main.go

migrate-up:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)" up

migrate-down:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)" down

seed:
	go run seed/seeder.go

test:
	go test ./... -v -short

test-e2e:
	go test ./test/e2e/... -v -tags=e2e

test-coverage:
	go test ./... -cover -short -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

swagger:
	swag init -g cmd/api/main.go -o docs

lint:
	golangci-lint run

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

clean:
	rm -rf bin/ coverage.out coverage.html
