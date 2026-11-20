COMPOSE := docker compose -f .development/docker-compose.yml

.PHONY: test up down bucketlimiter

test:
	go test ./...

# make up — поднять всё окружение
# make up bucketlimiter — поднять Redis для счётчиков
up:
ifeq ($(filter-out up,$(MAKECMDGOALS)),)
	$(COMPOSE) up -d
else
	@:
endif

# make down — остановить и удалить окружение
# make down bucketlimiter — остановить и удалить контейнер Redis
down:
ifeq ($(filter-out down,$(MAKECMDGOALS)),)
	$(COMPOSE) down
else
	@:
endif

bucketlimiter:
ifneq ($(filter down,$(MAKECMDGOALS)),)
	$(COMPOSE) rm --stop --force redis
else
	$(COMPOSE) up -d redis
endif
