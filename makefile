APP_NAME=tyrants-server
CMD_DIR=./cmd/server

.PHONY: run build clean tidy

run:
	go run $(CMD_DIR)

build:
	go build -o $(APP_NAME) $(CMD_DIR)

clean:
	rm -f $(APP_NAME)

tidy:
	go mod tidy

APP_NAME=tyrants-server
CMD_DIR=./cmd/server

.PHONY: run build clean tidy

run:
	go run $(CMD_DIR)

build:
	go build -o $(APP_NAME) $(CMD_DIR)

clean:
	rm -f $(APP_NAME)

tidy:
	go mod tidy


