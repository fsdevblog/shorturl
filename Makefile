# Название бинарного файла
BINARY=shortener

# Директория с main.go
CMD_DIR=cmd/shortener

# Бинарник с авто-тестами
TESTER=./cli/shortenertestbeta

# Порт для тестового сервера авто-тестов
SERVER_PORT=8080

# Переменная для номера теста
ITER?=1

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
	$(TESTER) -test.v -test.run=^TestIteration$(ITER)$$ -binary-path=$(CMD_DIR)/$(BINARY) -source-path=./ -server-port=$(SERVER_PORT)

# Запуск локальных тестов
test:
	go test ./... -v
