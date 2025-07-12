# Название бинарного файла
BINARY=shortener

# Директория с main.go
CMD_DIR=cmd/shortener

# Бинарник с авто-тестами
TESTER=./cli/shortenertestbeta

# Порт для тестового сервера авто-тестов
SERVER_PORT=8080

# Путь для файла бекапа
FILE_STORAGE_PATH=backup.json

DATABASE_DSN=postgres://study1-user:123123123@localhost:5435/postgres?sslmode=disable

# Переменная для номера теста
ITER?=1

# Переменная для имени миграции
MN?=new

dev-db-up:
	docker compose up -d postgres

dev-db-down:
	docker compose down --remove-orphans

dev-up:
	make dev-db-up
	make dev-run

dev-run:
	go run $(CMD_DIR)/main.go -a :$(SERVER_PORT) -d $(DATABASE_DSN)

dev-run-memory:
	go run $(CMD_DIR)/main.go -a :$(SERVER_PORT)

# Цель по умолчанию
all: build

# Билд приложения
build:
	cd $(CMD_DIR) && go build -o $(BINARY) *.go

# Очистка скомпилированного бинарника
clean:
	rm -f $(CMD_DIR)/$(BINARY)

# Пересобрать проект
rebuild: clean build

# Запуск автотестов с динамическим номером теста
auto-test: build
	$(TESTER) -test.v -test.run=^TestIteration$(ITER)$$ -binary-path=$(CMD_DIR)/$(BINARY) -source-path=./ -server-port=$(SERVER_PORT) -file-storage-path=$(FILE_STORAGE_PATH) -database-dsn=$(DATABASE_DSN)

# Запуск локальных тестов
test:
	go test ./... -v

# Создать миграцию
migrate-create:
	migrate create -ext sql -dir internal/db/migrations -seq ${MN}

# Миграция вверх
migrate-up:
	migrate -database "postgres://study1-user:123123123@localhost:5435/postgres?sslmode=disable" -path ./internal/db/migrations up 1

# Миграция вниз
migrate-down:
	migrate -database "postgres://study1-user:123123123@localhost:5435/postgres?sslmode=disable" -path ./internal/db/migrations down 1

# Запустить профилирование

pprof-profile:
	go tool pprof -http=":9090" -seconds=30 http://localhost:$(SERVER_PORT)/debug/pprof/profile

pprof-heap:
	go tool pprof -http=":9090" -seconds=30 http://localhost:$(SERVER_PORT)/debug/pprof/heap
