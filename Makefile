# Simple Makefile for common local development tasks

SHELL := /bin/bash

.PHONY: up down logs run build test fmt tidy env

# Load env from .env if present
ifneq (,$(wildcard ./.env))
include .env
export
endif

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

# Run the API with environment from .env (or overridden env vars)
run:
	@echo "Running API (HTTP_PORT=${HTTP_PORT})"
	POSTGRES_DSN=${POSTGRES_DSN} REDIS_ADDR=${REDIS_ADDR} HTTP_PORT=${HTTP_PORT} SERVICE_NAME=${SERVICE_NAME} SERVICE_VERSION=${SERVICE_VERSION} ENV=${ENV} go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	gofmt -w . && go test ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

env:
	@echo "Copy .env.example -> .env and edit values if needed"
	@echo "cat .env.example"
	@cat .env.example
