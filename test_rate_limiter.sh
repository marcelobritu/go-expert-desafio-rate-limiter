#!/bin/bash

# Script para testar o rate limiter
# Execute este script após iniciar o servidor

echo "=== Teste do Rate Limiter ==="
echo "Certifique-se de que o servidor está rodando em http://localhost:8080"
echo ""

# Função para fazer requisição e mostrar resultado
make_request() {
    local url=$1
    local headers=$2
    local description=$3
    
    echo "--- $description ---"
    if [ -n "$headers" ]; then
        response=$(curl -s -w "\nHTTP_CODE:%{http_code}\nTIME:%{time_total}" -H "$headers" "$url")
    else
        response=$(curl -s -w "\nHTTP_CODE:%{http_code}\nTIME:%{time_total}" "$url")
    fi
    
    echo "$response"
    echo ""
}

# Teste 1: Health check
make_request "http://localhost:8080/health" "" "Health Check"

# Teste 2: Informações de rate limit
make_request "http://localhost:8080/rate-limit/info" "" "Rate Limit Info"

# Teste 3: Requisições normais (limitação por IP)
echo "=== Testando Limitação por IP ==="
for i in {1..12}; do
    echo "Requisição $i:"
    make_request "http://localhost:8080/api/test" "" "API Test (IP Limit)"
    sleep 0.1
done

echo "Aguardando 2 segundos para reset do contador..."
sleep 2

# Teste 4: Teste com token
echo "=== Testando Limitação por Token ==="
for i in {1..5}; do
    echo "Requisição $i:"
    make_request "http://localhost:8080/api/test" "API_KEY: abc123" "API Test (Token)"
    sleep 0.1
done

# Teste 5: Reset de rate limit
echo "=== Testando Reset de Rate Limit ==="
make_request "http://localhost:8080/admin/reset/ip:127.0.0.1" "" "Reset IP Rate Limit"

# Teste 6: Verificar se reset funcionou
make_request "http://localhost:8080/api/test" "" "API Test (After Reset)"

echo "=== Teste Concluído ==="
echo "Verifique os headers de resposta para informações de rate limiting"
echo "Status 429 indica que o rate limit foi excedido"
