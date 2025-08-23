
all: run

local:
	go run cmd/wbOrderSaver.go -env local

run:
	docker-compose up -d

stop:
	docker-compose stop

build:
	docker-compose build

reduild:
	docker-compose up -d --build wbordersaver

restart:
	docker-compose down && docker-compose up -d

clean:
	docker-compose down -v && docker-compose up -d
