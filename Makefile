# Makefile para o Rate Limiter (go-chi)

.PHONY: help build run test clean docker-up docker-down

# Variáveis
BINARY_NAME=rate-limiter
DOCKER_COMPOSE=docker-compose

# Ajuda
help: ## Mostra esta ajuda
	@echo "Comandos disponíveis:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Desenvolvimento
build: ## Compila o projeto
	go build -o $(BINARY_NAME) cmd/server/main.go

run: ## Executa o servidor
	go run cmd/server/main.go

test: ## Executa os testes
	go test ./...

# Docker
docker-up: ## Inicia o Redis com Docker Compose
	$(DOCKER_COMPOSE) up -d redis

docker-down: ## Para o Redis
	$(DOCKER_COMPOSE) down

docker-logs: ## Mostra logs do Redis
	$(DOCKER_COMPOSE) logs -f redis

# Limpeza
clean: ## Remove arquivos compilados
	go clean
	rm -f $(BINARY_NAME)

# Dependências
deps: ## Baixa dependências
	go mod tidy
	go mod download

# Teste do rate limiter
test-rate-limiter: ## Executa script de teste do rate limiter
	./test_rate_limiter.sh

# Setup completo
setup: deps docker-up ## Configura o ambiente completo
	@echo "Ambiente configurado!"
	@echo "Execute 'make run' para iniciar o servidor"
	@echo "Execute 'make test-rate-limiter' para testar"

# Verificação
check: ## Verifica o código
	go vet ./...
	go fmt ./...

# Instalação
install: build ## Instala o binário
	sudo cp $(BINARY_NAME) /usr/local/bin/

# Desinstalação
uninstall: ## Remove o binário
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
