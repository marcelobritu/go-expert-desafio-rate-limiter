# Rate Limiter em Go

Um rate limiter robusto e configurável em Go que pode limitar requisições por endereço IP ou token de acesso, com suporte a diferentes estratégias de armazenamento.

## Características

- ✅ **Limitação por IP**: Controle de requisições baseado no endereço IP do cliente
- ✅ **Limitação por Token**: Controle de requisições baseado em tokens de acesso únicos
- ✅ **Priorização de Token**: Tokens configurados sobrescrevem limites de IP
- ✅ **Estratégia de Armazenamento**: Interface flexível para diferentes mecanismos de persistência
- ✅ **Redis Support**: Implementação completa com Redis para alta performance
- ✅ **Configuração via Ambiente**: Configuração através de variáveis de ambiente ou arquivo .env
- ✅ **Middleware go-chi**: Integração fácil com servidores web go-chi
- ✅ **Headers Informativos**: Headers HTTP com informações de rate limiting
- ✅ **Bloqueio Temporário**: Bloqueio configurável quando limites são excedidos

## Arquitetura

```
├── config/          # Configuração com Viper
├── strategy/        # Interface e implementações de armazenamento
├── limiter/         # Lógica principal do rate limiter
├── middleware/      # Middleware para integração com go-chi
├── cmd/server/      # Servidor de exemplo
└── docker-compose.yml
```

## Instalação e Configuração

### 1. Pré-requisitos

- Go 1.25.1 ou superior
- Docker e Docker Compose (para Redis)

### 2. Clone e Dependências

```bash
git clone <repository-url>
cd go-expert-desafio-rate-limiter
go mod tidy
```

### 3. Configuração do Redis

```bash
# Iniciar Redis com Docker Compose
docker-compose up -d redis

# Verificar se Redis está rodando
docker-compose ps
```

### 4. Configuração de Ambiente

Copie o arquivo de exemplo e configure suas variáveis:

```bash
cp env.example .env
```

Edite o arquivo `.env` com suas configurações:

```env
# Server Configuration
SERVER_PORT=8080

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Rate Limiting Configuration
RATE_LIMIT_IP_LIMIT=10
RATE_LIMIT_IP_BLOCK_TIME=1m

# Token-specific rate limits (opcional)
RATE_LIMIT_TOKEN_ABC123_LIMIT=100
RATE_LIMIT_TOKEN_ABC123_BLOCK_TIME=5m
RATE_LIMIT_TOKEN_PREMIUM_LIMIT=1000
RATE_LIMIT_TOKEN_PREMIUM_BLOCK_TIME=10m
```

### 5. Executar o Servidor

```bash
go run cmd/server/main.go
```

O servidor estará disponível em `http://localhost:8080`

## Uso

### Endpoints Disponíveis

- `GET /health` - Health check (sem rate limiting)
- `GET /rate-limit/info` - Informações de rate limit (sem incrementar contador)
- `GET /api/test` - Endpoint protegido para teste
- `POST /api/data` - Endpoint POST protegido
- `GET /api/status` - Status da API com informações de rate limit
- `POST /admin/reset/:key` - Reset de rate limit para uma chave específica

### Exemplos de Uso

#### 1. Teste Básico (Limitação por IP)

```bash
# Fazer requisições para testar o rate limiting
curl http://localhost:8080/api/test

# Verificar headers de rate limit
curl -I http://localhost:8080/api/test
```

#### 2. Teste com Token

```bash
# Usar token configurado
curl -H "API_KEY: abc123" http://localhost:8080/api/test

# Token premium com limite maior
curl -H "API_KEY: premium" http://localhost:8080/api/test
```

#### 3. Verificar Informações de Rate Limit

```bash
# Ver informações sem incrementar contador
curl http://localhost:8080/rate-limit/info
```

#### 4. Reset de Rate Limit

```bash
# Reset para um IP específico
curl -X POST http://localhost:8080/admin/reset/ip:192.168.1.1

# Reset para um token específico
curl -X POST http://localhost:8080/admin/reset/token:abc123
```

### Headers de Resposta

O rate limiter adiciona os seguintes headers HTTP:

- `X-RateLimit-Remaining`: Número de requisições restantes
- `X-RateLimit-Reset`: Timestamp de quando o contador será resetado
- `X-RateLimit-Block-Time`: Tempo de bloqueio (quando aplicável)
- `X-RateLimit-Count`: Contador atual (apenas no endpoint /rate-limit/info)

### Resposta de Rate Limit Excedido

Quando o limite é excedido, o servidor retorna:

```json
{
  "error": "Rate limit exceeded",
  "message": "you have reached the maximum number of requests or actions allowed within a certain time frame",
  "details": {
    "reason": "IP rate limit exceeded",
    "reset_time": "2024-01-01T12:00:00Z",
    "block_time": "1m0s"
  }
}
```

Status HTTP: `429 Too Many Requests`

## Configuração Avançada

### Tokens Personalizados

Para configurar tokens específicos, adicione variáveis de ambiente no formato:

```env
RATE_LIMIT_TOKEN_<TOKEN_NAME>_LIMIT=<limit>
RATE_LIMIT_TOKEN_<TOKEN_NAME>_BLOCK_TIME=<duration>
```

Exemplos:

```env
# Token básico
RATE_LIMIT_TOKEN_BASIC_LIMIT=50
RATE_LIMIT_TOKEN_BASIC_BLOCK_TIME=2m

# Token premium
RATE_LIMIT_TOKEN_PREMIUM_LIMIT=1000
RATE_LIMIT_TOKEN_PREMIUM_BLOCK_TIME=10m

# Token de teste
RATE_LIMIT_TOKEN_ABC123_LIMIT=100
RATE_LIMIT_TOKEN_ABC123_BLOCK_TIME=5m
```

### Integração com Seu Projeto

Para usar o rate limiter em seu próprio projeto:

```go
package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/marcelobritu/go-expert-desafio-rate-limiter/config"
    "github.com/marcelobritu/go-expert-desafio-rate-limiter/limiter"
    ratelimitMiddleware "github.com/marcelobritu/go-expert-desafio-rate-limiter/middleware"
    "github.com/marcelobritu/go-expert-desafio-rate-limiter/strategy"
)

func main() {
    // Carregar configuração
    cfg, _ := config.LoadConfig()
    
    // Inicializar Redis
    redisStrategy := strategy.NewRedisStrategy(
        cfg.Redis.Host,
        cfg.Redis.Port,
        cfg.Redis.Password,
        cfg.Redis.DB,
    )
    
    // Criar rate limiter
    rateLimiter := limiter.NewRateLimiter(redisStrategy, cfg)
    
    // Configurar go-chi
    router := chi.NewRouter()
    router.Use(middleware.Logger)
    router.Use(middleware.Recoverer)
    router.Use(ratelimitMiddleware.RateLimitMiddleware(rateLimiter))
    
    // Suas rotas aqui
    router.Get("/api/your-endpoint", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"message": "Hello World"}`))
    })
    
    http.ListenAndServe(":8080", router)
}
```

## Estratégias de Armazenamento

O projeto implementa o padrão Strategy para permitir diferentes mecanismos de armazenamento:

### Interface Strategy

```go
type StorageStrategy interface {
    Get(ctx context.Context, key string) (*RateLimitInfo, error)
    Set(ctx context.Context, key string, info *RateLimitInfo, expiration time.Duration) error
    Increment(ctx context.Context, key string, expiration time.Duration) (int, error)
    SetBlocked(ctx context.Context, key string, blockUntil time.Time) error
    IsBlocked(ctx context.Context, key string) (bool, time.Time, error)
    Delete(ctx context.Context, key string) error
    Close() error
}
```

### Implementação Redis

A implementação Redis atual suporta:
- Operações atômicas com pipeline
- Expiração automática de chaves
- Bloqueio temporário
- Persistência de dados

### Adicionando Novas Estratégias

Para adicionar uma nova estratégia (ex: Memcached, In-Memory):

1. Implemente a interface `StorageStrategy`
2. Adicione configuração para a nova estratégia
3. Modifique o factory de criação de estratégias

## Testes

### Teste de Carga

Para testar o rate limiter sob carga:

```bash
# Instalar hey (ferramenta de teste de carga)
go install github.com/rakyll/hey@latest

# Teste básico
hey -n 100 -c 10 http://localhost:8080/api/test

# Teste com token
hey -n 200 -c 20 -H "API_KEY: abc123" http://localhost:8080/api/test
```

### Teste Manual

```bash
# Script para testar rate limiting
for i in {1..15}; do
  echo "Request $i:"
  curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" http://localhost:8080/api/test
  sleep 0.1
done
```

## Monitoramento

### Redis Commander

O docker-compose inclui Redis Commander para monitoramento:

```bash
# Acessar Redis Commander
open http://localhost:8081
```

### Logs

O servidor registra:
- Conexão com Redis
- Erros de rate limiting
- Informações de configuração

## Troubleshooting

### Problemas Comuns

1. **Redis não conecta**: Verifique se o Redis está rodando com `docker-compose ps`
2. **Rate limit não funciona**: Verifique as configurações no arquivo `.env`
3. **Tokens não reconhecidos**: Verifique o formato das variáveis de ambiente

### Debug

Para debug, verifique:
- Logs do servidor
- Conexão Redis com `redis-cli ping`
- Configurações carregadas no endpoint `/health`

## Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature
3. Commit suas mudanças
4. Push para a branch
5. Abra um Pull Request

## Licença

Este projeto está sob a licença MIT. Veja o arquivo LICENSE para mais detalhes.
