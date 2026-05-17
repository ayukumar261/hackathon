.PHONY: up down logs restart clean migrate

up:
	docker compose up --build

migrate:
	docker compose exec go-api go run ./cmd/migrate

down:
	docker compose down

logs:
	docker compose logs -f

restart:
	docker compose restart

clean:
	docker compose down -v
