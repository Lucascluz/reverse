# Makefile for reverxy reverxy proxy

CONFIG_PATH="./config.yaml"

BINARY_NAME=reverxy
CMD_DIR=cmd
MAIN_FILE=$(CMD_DIR)/main.go
DOCKER_IMAGE=lcluz/reverxy:latest

.PHONY: build run clean docker-build docker-push test

build:
	go build -o ./bin/$(BINARY_NAME) $(MAIN_FILE)

CONFIG_PATH ?= /etc/config/config.yaml

run: build
	CONFIG_PATH=$(CONFIG_PATH) ./bin/$(BINARY_NAME)

clean:
	rm -f ./bin/$(BINARY_NAME)

mod-tidy:
	go mod tidy

docker-build:
	docker build -t $(DOCKER_IMAGE) .

docker-push: docker-build
	docker push $(DOCKER_IMAGE)

test:
	CONFIG_PATH=$(CONFIG_PATH) go test ./...

fmt:
	gofmt -w .

lint:
	golint ./...
