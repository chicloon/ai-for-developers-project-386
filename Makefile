# Makefile for Call Booking Project
# Monorepo with Go backend, Next.js frontend, TypeSpec API contracts

.PHONY: help install build test dev clean docker-up docker-down typespec

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(BLUE)Call Booking Project - Available Commands:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

# Installation targets
install: install-typespec install-web ## Install all dependencies

install-typespec: ## Install TypeSpec dependencies
	@echo "$(YELLOW)Installing TypeSpec dependencies...$(NC)"
	cd typespec && npm install

install-web: ## Install Next.js frontend dependencies
	@echo "$(YELLOW)Installing Next.js dependencies...$(NC)"
	cd web && npm install

install-go: ## Download Go dependencies
	@echo "$(YELLOW)Downloading Go dependencies...$(NC)"
	go mod download

# Build targets
build: build-go build-web ## Build all components

build-go: ## Build Go backend server
	@echo "$(YELLOW)Building Go server...$(NC)"
	go build -o bin/server ./cmd/server

build-web: ## Build Next.js frontend for production
	@echo "$(YELLOW)Building Next.js frontend...$(NC)"
	cd web && npm run build

# Development targets
dev: ## Start development environment with Docker Compose
	@echo "$(YELLOW)Starting development environment...$(NC)"
	docker compose up -d

dev-down: ## Stop development environment
	@echo "$(YELLOW)Stopping development environment...$(NC)"
	docker compose down

dev-logs: ## Show logs from development environment
	docker compose logs -f

# Testing targets
test: test-go test-web ## Run all tests

test-go: ## Run Go tests
	@echo "$(YELLOW)Running Go tests...$(NC)"
	go test ./... -v

test-web: ## Run Next.js tests
	@echo "$(YELLOW)Running Next.js tests...$(NC)"
	cd web && npm test

test-e2e: ## Run Playwright E2E tests
	@echo "$(YELLOW)Running E2E tests...$(NC)"
	cd web && npx playwright install chromium
	cd web && npm run test:e2e

# TypeSpec targets
typespec: ## Compile TypeSpec to OpenAPI
	@echo "$(YELLOW)Compiling TypeSpec...$(NC)"
	cd typespec && npx tsp compile .

typespec-watch: ## Compile TypeSpec in watch mode
	@echo "$(YELLOW)Compiling TypeSpec (watch mode)...$(NC)"
	cd typespec && npx tsp compile . --watch

# Code quality
lint: lint-go lint-web ## Run all linters

lint-go: ## Run Go linter
	@echo "$(YELLOW)Running Go linter...$(NC)"
	golangci-lint run ./...

lint-web: ## Run Next.js linter
	@echo "$(YELLOW)Running Next.js linter...$(NC)"
	cd web && npm run lint

fmt: fmt-go fmt-web ## Format all code

fmt-go: ## Format Go code
	@echo "$(YELLOW)Formatting Go code...$(NC)"
	gofmt -w ./internal ./cmd

fmt-web: ## Format Next.js code
	@echo "$(YELLOW)Formatting Next.js code...$(NC)"
	cd web && npm run lint -- --fix

# Database targets
migrate: ## Run database migrations (requires running DB)
	@echo "$(YELLOW)Running database migrations...$(NC)"
	go run ./cmd/server migrate

# Docker targets
docker-build: ## Build all Docker images
	@echo "$(YELLOW)Building Docker images...$(NC)"
	docker compose build

docker-push: ## Push Docker images to registry
	@echo "$(YELLOW)Pushing Docker images...$(NC)"
	docker compose push

# Clean targets
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf bin/
	rm -rf web/.next/
	rm -rf web/node_modules/
	rm -rf typespec/node_modules/

clean-docker: ## Clean Docker containers, volumes, and images
	@echo "$(YELLOW)Cleaning Docker resources...$(NC)"
	docker compose down -v --rmi all

# Utilities
start-server: build-go ## Build and start Go server locally
	@echo "$(YELLOW)Starting Go server...$(NC)"
	./bin/server

start-web: ## Start Next.js development server
	@echo "$(YELLOW)Starting Next.js dev server...$(NC)"
	cd web && npm run dev

ps: ## Show running containers
	docker compose ps
