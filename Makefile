
all: run

local:
	go run cmd/bot.go -env local

run:
	docker compose -p kinopoisktwoactors up -d

stop:
	docker compose -p kinopoisktwoactors stop

build:
	docker compose build

restart:
	docker compose down && docker compose up -d

clean:
