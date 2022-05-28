SHELL := /bin/bash
PWD := $(shell pwd)

GIT_REMOTE = github.com/7574-sistemas-distribuidos/docker-compose-init

default: build

all:

deps:
	go mod tidy
	go mod vendor

build: deps
	GOOS=linux go build -o bin/client github.com/7574-sistemas-distribuidos/docker-compose-init/client
.PHONY: build

docker-purge:
	docker rmi `docker images --filter label=intermediateStageToBeDeleted=true -q`
.PHONY: docker-purge

docker-image:
	docker build -f ./server/admin/Dockerfile -t "consumer:latest" .
	docker build -f ./producer/Dockerfile -t "producer:latest" .
	# Execute this command from time to time to clean up intermediate stages generated 
	# during client build (your hard drive will like this :) ). Don't left uncommented if you 
	# want to avoid rebuilding client image every time the docker-compose-up command 
	# is executed, even when client code has not changed
	docker rmi `docker images --filter label=intermediateStageToBeDeleted=true -q`
.PHONY: docker-image

docker-server-image:
	docker build -f ./server/Dockerfile -t "server:latest" .
.PHONY: docker-server-image

docker-client-image:
	docker build -f ./client/Dockerfile -t "client:latest" .
.PHONY: docker-client-app-image

server-up:
	docker-compose -f docker-compose-server.yaml up --build
.PHONY: server-up

client-up:
	docker-compose -f docker-compose-client.yaml up --build
.PHONY: client-app-up

server-down:
	docker-compose -f docker-compose-server.yaml stop -t 10
	docker-compose -f docker-compose-server.yaml down --remove-orphans
.PHONY: server-down

client-down:
	docker-compose -f docker-compose-client.yaml stop -t 10
	docker-compose -f docker-compose-client.yaml down
.PHONY: client-app-down

server-logs:
	docker-compose -f docker-compose-server.yaml logs -f
.PHONY: server-logs

client-logs:
	docker-compose -f docker-compose-clients.yaml logs -f
.PHONY: client-app-logs
