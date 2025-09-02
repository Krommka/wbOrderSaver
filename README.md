# WB L0 Order Service
Микросервис для управления заказами с Kafka-интеграцией, PostgreSQL хранилищем, Redis кэш хранилищем и веб-интерфейсом.
## Быстрый старт

1. Подготовьте файл `.env` (используйте `example.env` как образец)
2. Запустите сборку инфраструктуры сервиса: `make infra`
3. Через **Kafka UI**: http://localhost:9020 создайте топик (`Orders` в .env по умолчанию) с необходимыми настройками
4. Запустите сборку самого сервиса: `make app`
5. Для запуска скрипта создания заказов, выполните: `make orders`

##  Управление сервисом

| Команда        | Описание                     |
|----------------|------------------------------|
| `make`         | Запуск всего проекта         |
| `make infra`   | Запуск инфраструктуры        |
| `make app`     | Запуск сервиса               |
| `make orders`  | Создание заказов             |
| `make rebuild` | Быстрое обновление сервиса   |
| `make restart` | Перезапуск сервиса           |
| `make clean`   | Очистка volume в контейнерах |
| `make test`    | Запуск тестов                |

## Конфигурация

- **`REDIS_CAPACITY=<int>`** - количество заказов, хранимых в кэше
- **`REDIS_WARMUP=true`** - прогрев кэша при запуске

## Доступные интерфейсы

| Сервис             | URL |
|--------------------|-----|
| **Kafka UI**       | http://localhost:9020 |
| **Healthcheck**    | http://localhost:8081/api/v1/health |
| **Web Interface**  | http://localhost:8081 |
| **Get Order JSON** | http://localhost:8081/order/<order_uid> |
| **Swagger Docs**   | http://localhost:8081/swagger/index.html |
