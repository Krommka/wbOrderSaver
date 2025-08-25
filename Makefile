
all: run

local:
	go run cmd/wbOrderSaver/main.go -env local

run:
	docker-compose up -d

stop:
	docker-compose stop

build:
	docker-compose build

rebuild:
	docker-compose up -d --build wbordersaver

restart:
	docker-compose down && docker-compose up -d

clean:
	docker-compose down -v && docker-compose up -d


test:
	go test ./internal/repository/postgres ./internal/usecase -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
