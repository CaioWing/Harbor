# Harbor
 
Sistema de deploy de arquivos individuais (single-file OTA) para dispositivos remotos. Substituto do Mender focado em simplicidade, eficiencia e granularidade por arquivo.
 
Diferente do Mender (que trabalha com imagens rootfs completas), o Harbor opera no nivel de arquivo individual, permitindo atualizar um unico binario, config ou script sem re-flashar o sistema inteiro.
 
## Indice
 
- [Arquitetura](#arquitetura)
- [Quick Start](#quick-start)
- [Swagger / OpenAPI](#swagger--openapi)
- [Configuracao do Servidor](#configuracao-do-servidor)
- [API de Gerenciamento (Frontend/Operador)](#api-de-gerenciamento)
- [Configurando um Device (Client)](#configurando-um-device)
- [Fluxo Completo: Deploy de um Arquivo](#fluxo-completo)
- [Variaveis de Ambiente](#variaveis-de-ambiente)
- [Desenvolvimento](#desenvolvimento)
 
---
 
## Arquitetura
 
```
┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│  Frontend    │────>│  Harbor Server   │<────│  Device      │
│  (React)     │     │  (Go API)        │     │  (Agent)     │
│              │     │                  │     │              │
│  Management  │     │  - PostgreSQL    │     │  - Polling   │
│  API (JWT)   │     │  - File Storage  │     │  - Download  │
└──────────────┘     └──────────────────┘     └──────────────┘
```
 
O sistema possui duas APIs separadas:
 
- **Management API** (`/api/v1/management`) — usada pelo frontend React ou por operadores via curl. Autenticacao via JWT.
- **Device API** (`/api/v1/device`) — usada pelos dispositivos remotos. Autenticacao via token opaco.
 
### Stack
 
| Componente     | Tecnologia                   |
|----------------|------------------------------|
| Linguagem      | Go 1.22+                     |
| HTTP Router    | chi                          |
| Banco de Dados | PostgreSQL 16                |
| Driver DB      | pgx/v5                       |
| Migrations     | golang-migrate (embedded)    |
| Auth           | JWT (golang-jwt) + tokens    |
| Logging        | slog (stdlib)                |
| File Storage   | Filesystem local             |
 
---
 
## Quick Start
 
### Com Docker Compose (recomendado)
 
```bash
# Sobe o servidor + PostgreSQL
make docker-up
 
# Acompanhe os logs
make docker-logs
```
 
O servidor estara disponivel em `http://localhost:8080`.
 
### Sem Docker
 
Requisitos: Go 1.22+, PostgreSQL 16 rodando.
 
```bash
# Configure as variaveis de ambiente (ou use os defaults)
export HARBOR_DB_HOST=localhost
export HARBOR_DB_PASSWORD=harbor
export HARBOR_JWT_SECRET=minha-chave-secreta
export HARBOR_STORAGE_PATH=./data/artifacts
 
# Build e execucao
make run
```
 
### Verificar se esta rodando
 
```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### Swagger / OpenAPI

A documentacao da API fica disponivel em:

- **Swagger UI:** `http://localhost:8080/docs`
- **Spec OpenAPI (YAML):** `http://localhost:8080/docs/openapi.yaml`

---
 
## Configuracao do Servidor
 
Toda configuracao e feita via variaveis de ambiente. Veja a secao [Variaveis de Ambiente](#variaveis-de-ambiente) para a lista completa.
 
Para producao, voce **deve** alterar:
 
```bash
HARBOR_JWT_SECRET=uma-chave-forte-e-aleatoria
HARBOR_DB_PASSWORD=senha-segura
HARBOR_CORS_ORIGINS=https://seu-frontend.com
```
 
---
 
## API de Gerenciamento
 
A Management API e usada pelo frontend React ou diretamente via `curl` para gerenciar dispositivos, artifacts e deployments.
 
### Autenticacao
 
```bash
# Login (credenciais default: admin@harbor.local / admin)
curl -X POST http://localhost:8080/api/v1/management/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@harbor.local", "password": "admin"}'
 
# Resposta:
# {"token": "eyJhbG...", "expires_at": "2026-02-17T12:00:00Z"}
```
 
Use o token retornado em todas as chamadas seguintes:
 
```bash
TOKEN="eyJhbG..."
```
 
Para renovar o token sem fazer login novamente:
 
```bash
curl -X POST http://localhost:8080/api/v1/management/auth/refresh \
  -H "Authorization: Bearer $TOKEN"
```
 
### Gerenciamento de Devices
 
**Listar devices** (com filtros e paginacao):
 
```bash
curl http://localhost:8080/api/v1/management/devices \
  -H "Authorization: Bearer $TOKEN"
 
# Filtros disponiveis:
#   ?status=pending          (pending, accepted, rejected, decommissioned)
#   ?device_type=raspberry-pi-4
#   ?tag=production
#   ?page=1&per_page=20
#   ?sort=created_at&order=desc
```
 
**Aceitar um device pendente:**
 
```bash
curl -X PUT http://localhost:8080/api/v1/management/devices/{device_id}/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "accepted"}'
```
 
**Adicionar tags a um device:**
 
```bash
curl -X PATCH http://localhost:8080/api/v1/management/devices/{device_id}/tags \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tags": ["production", "rack-01", "sp"]}'
```
 
**Contagem de devices por status:**
 
```bash
curl http://localhost:8080/api/v1/management/devices/count \
  -H "Authorization: Bearer $TOKEN"
 
# {"pending": 3, "accepted": 10, "rejected": 1}
```
 
### Upload de Artifacts
 
Um artifact e um arquivo individual (binario, config, script) que sera deployado nos devices.
 
```bash
curl -X POST http://localhost:8080/api/v1/management/artifacts \
  -H "Authorization: Bearer $TOKEN" \
  -F "name=myapp" \
  -F "version=1.2.0" \
  -F "description=Nova versao do aplicativo" \
  -F "target_path=/usr/local/bin/myapp" \
  -F "file_mode=0755" \
  -F "file_owner=root:root" \
  -F "device_types=raspberry-pi-4,beaglebone" \
  -F "pre_install_cmd=systemctl stop myapp" \
  -F "post_install_cmd=systemctl start myapp" \
  -F "file=@./build/myapp"
```
 
Campos do artifact:
 
| Campo            | Obrigatorio | Descricao                                    |
|------------------|-------------|----------------------------------------------|
| `name`           | Sim         | Nome do artifact (ex: `myapp`)               |
| `version`        | Sim         | Versao (ex: `1.2.0`)                         |
| `target_path`    | Sim         | Caminho de destino no device                 |
| `device_types`   | Sim         | Tipos de device compativeis (separados por `,`) |
| `file`           | Sim         | O arquivo em si (max 500MB)                  |
| `file_mode`      | Nao         | Permissoes Unix (default: `0644`)            |
| `file_owner`     | Nao         | Dono do arquivo (ex: `root:root`)            |
| `description`    | Nao         | Descricao livre                              |
| `pre_install_cmd`| Nao         | Comando executado antes da instalacao        |
| `post_install_cmd`| Nao        | Comando executado depois da instalacao       |
| `rollback_cmd`   | Nao         | Comando executado em caso de falha           |
 
### Criar Deployments
 
Um deployment envia um artifact para um conjunto de devices.
 
```bash
# Deploy para devices especificos por ID
curl -X POST http://localhost:8080/api/v1/management/deployments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "deploy-myapp-v1.2.0",
    "artifact_id": "uuid-do-artifact",
    "target_device_ids": ["uuid-device-1", "uuid-device-2"]
  }'
 
# Deploy para todos os devices com determinadas tags
curl -X POST http://localhost:8080/api/v1/management/deployments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "deploy-producao",
    "artifact_id": "uuid-do-artifact",
    "target_device_tags": ["production"]
  }'
 
# Deploy para todos os devices de um tipo
curl -X POST http://localhost:8080/api/v1/management/deployments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "deploy-raspberries",
    "artifact_id": "uuid-do-artifact",
    "target_device_types": ["raspberry-pi-4"]
  }'
```
 
**Acompanhar status do deployment:**
 
```bash
# Status geral
curl http://localhost:8080/api/v1/management/deployments/{deployment_id} \
  -H "Authorization: Bearer $TOKEN"
 
# Status por device
curl http://localhost:8080/api/v1/management/deployments/{deployment_id}/devices \
  -H "Authorization: Bearer $TOKEN"
 
# Estatisticas gerais
curl http://localhost:8080/api/v1/management/deployments/statistics \
  -H "Authorization: Bearer $TOKEN"
```
 
**Cancelar um deployment:**
 
```bash
curl -X POST http://localhost:8080/api/v1/management/deployments/{deployment_id}/cancel \
  -H "Authorization: Bearer $TOKEN"
```
 
### Rollback
 
Para fazer rollback, basta criar um novo deployment apontando para o artifact da versao anterior. O Harbor mantem historico de todos os artifacts.
 
```bash
# Exemplo: rollback do myapp para versao 1.1.0
curl -X POST http://localhost:8080/api/v1/management/deployments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rollback-myapp-v1.1.0",
    "artifact_id": "uuid-do-artifact-v1.1.0",
    "target_device_tags": ["production"]
  }'
```
 
### Audit Log
 
Todas as acoes de gerenciamento (aceitar device, upload de artifact, criar deployment, etc.) sao registradas automaticamente.
 
```bash
curl http://localhost:8080/api/v1/management/audit \
  -H "Authorization: Bearer $TOKEN"
 
# Filtros:
#   ?actor=admin
#   ?action=device.update_status
#   ?resource=deployment
#   ?page=1&per_page=20
#   ?order=desc
```
 
---
 
## Configurando um Device
 
O device (client) se comunica com o Harbor via polling HTTP. Abaixo esta o fluxo completo e exemplos de como implementar um client.
 
### Ciclo de Vida do Device
 
```
[novo] ──> PENDING ──> ACCEPTED ──> ACTIVE (polling)
                   \──> REJECTED
 
ACTIVE ──> DECOMMISSIONED (removido permanentemente)
```
 
### 1. Registro Inicial (Admissao)
 
O device envia sua identidade para o servidor. No primeiro contato, o device fica com status `pending` ate que um operador o aceite.
 
```bash
HARBOR_URL="http://seu-servidor:8080"
 
# Registrar o device
curl -X POST $HARBOR_URL/api/v1/device/auth \
  -H "Content-Type: application/json" \
  -d '{
    "identity": {
      "mac_address": "aa:bb:cc:dd:ee:ff",
      "device_type": "raspberry-pi-4",
      "serial_number": "SN-2024-001"
    }
  }'
 
# Primeira resposta (device pendente):
# HTTP 401 {"error": "device is pending approval"}
```
 
O campo `device_type` e **obrigatorio**. Os demais campos de identidade sao livres.
 
A identidade gera um hash SHA-256 deterministico — mesmo que o device reinicie ou reinstale o agent, ele sera reconhecido pelo mesmo hash.
 
### 2. Aprovacao pelo Operador
 
O operador aceita o device via Management API:
 
```bash
curl -X PUT $HARBOR_URL/api/v1/management/devices/{device_id}/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "accepted"}'
```
 
### 3. Obtendo o Token
 
Apos o device ser aceito, a proxima chamada de auth retorna um token:
 
```bash
curl -X POST $HARBOR_URL/api/v1/device/auth \
  -H "Content-Type: application/json" \
  -d '{
    "identity": {
      "mac_address": "aa:bb:cc:dd:ee:ff",
      "device_type": "raspberry-pi-4",
      "serial_number": "SN-2024-001"
    }
  }'
 
# Resposta (device aceito):
# {"token": "a1b2c3d4e5f6..."}
```
 
O device deve armazenar este token localmente e usa-lo em todas as chamadas seguintes.
 
### 4. Reportar Inventory
 
O device pode reportar atributos dinamicos (OS, arquitetura, IP, etc.):
 
```bash
DEVICE_TOKEN="a1b2c3d4e5f6..."
 
curl -X PATCH $HARBOR_URL/api/v1/device/inventory \
  -H "Authorization: Bearer $DEVICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "os": "Linux 5.15",
    "arch": "arm64",
    "hostname": "edge-node-03",
    "ip_address": "192.168.1.50",
    "harbor_version": "0.1.0",
    "free_disk": "2.3GB",
    "location": "SP",
    "environment": "production"
  }'
```
 
### 5. Polling de Deployments
 
O device faz polling periodico para verificar se ha deployments pendentes:
 
```bash
curl $HARBOR_URL/api/v1/device/deployments/next \
  -H "Authorization: Bearer $DEVICE_TOKEN"
 
# Se houver deployment pendente:
# {
#   "deployment_id": "uuid",
#   "dd_id": "uuid",
#   "artifact": {
#     "name": "myapp",
#     "version": "1.2.0",
#     "target_path": "/usr/local/bin/myapp",
#     "file_mode": "0755",
#     "checksum_sha256": "abc123...",
#     "file_size": 1048576,
#     "download_url": "/api/v1/device/deployments/{dd_id}/download",
#     "pre_install_cmd": "systemctl stop myapp",
#     "post_install_cmd": "systemctl start myapp"
#   },
#   "retry": {
#     "max_attempts": 3,
#     "interval_sec": 30,
#     "backoff_multiplier": 2
#   }
# }
 
# Se nao houver nada pendente:
# HTTP 204 No Content
```
 
### 6. Download e Instalacao
 
Quando o device recebe um deployment, ele deve:
 
```bash
DD_ID="uuid-do-deployment-device"
 
# 1. Reportar que esta baixando
curl -X PUT $HARBOR_URL/api/v1/device/deployments/$DD_ID/status \
  -H "Authorization: Bearer $DEVICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "downloading", "log": "Iniciando download..."}'
 
# 2. Fazer download do arquivo
curl -o /tmp/myapp $HARBOR_URL/api/v1/device/deployments/$DD_ID/download \
  -H "Authorization: Bearer $DEVICE_TOKEN"
# Header X-Checksum-SHA256 retorna o checksum esperado
 
# 3. Verificar checksum SHA-256
echo "abc123...  /tmp/myapp" | sha256sum -c -
 
# 4. Reportar que esta instalando
curl -X PUT $HARBOR_URL/api/v1/device/deployments/$DD_ID/status \
  -H "Authorization: Bearer $DEVICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "installing", "log": "Verificacao OK. Instalando..."}'
 
# 5. Executar pre_install_cmd, mover arquivo, executar post_install_cmd
systemctl stop myapp
cp /tmp/myapp /usr/local/bin/myapp
chmod 0755 /usr/local/bin/myapp
systemctl start myapp
 
# 6. Reportar sucesso
curl -X PUT $HARBOR_URL/api/v1/device/deployments/$DD_ID/status \
  -H "Authorization: Bearer $DEVICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "success", "log": "Download OK. Checksum verificado. Servico reiniciado."}'
 
# Ou em caso de falha:
curl -X PUT $HARBOR_URL/api/v1/device/deployments/$DD_ID/status \
  -H "Authorization: Bearer $DEVICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "failure", "log": "Checksum invalido. Abortando."}'
```
 
### Exemplo: Agent Minimo (Shell Script)
 
Abaixo um agent minimo que pode rodar como servico em qualquer device Linux. Requer `curl` e `jq`.
 
```bash
#!/bin/bash
# harbor-agent.sh — Agent minimo para Harbor
 
HARBOR_URL="${HARBOR_URL:-http://harbor-server:8080}"
TOKEN_FILE="/etc/harbor/token"
POLL_INTERVAL="${POLL_INTERVAL:-60}"
 
# Identidade do device — ajuste conforme o hardware
MAC=$(cat /sys/class/net/eth0/address 2>/dev/null || echo "00:00:00:00:00:00")
DEVICE_TYPE="${DEVICE_TYPE:-linux-generic}"
IDENTITY="{\"identity\":{\"mac_address\":\"$MAC\",\"device_type\":\"$DEVICE_TYPE\"}}"
 
authenticate() {
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$HARBOR_URL/api/v1/device/auth" \
        -H "Content-Type: application/json" -d "$IDENTITY")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
 
    if [ "$HTTP_CODE" = "200" ]; then
        echo "$BODY" | jq -r '.token' > "$TOKEN_FILE"
        echo "[harbor] Autenticado com sucesso"
        return 0
    elif [ "$HTTP_CODE" = "401" ]; then
        echo "[harbor] Device pendente de aprovacao"
        return 1
    else
        echo "[harbor] Erro na autenticacao: $HTTP_CODE"
        return 1
    fi
}
 
report_status() {
    local dd_id=$1 status=$2 log_msg=$3
    curl -s -X PUT "$HARBOR_URL/api/v1/device/deployments/$dd_id/status" \
        -H "Authorization: Bearer $(cat $TOKEN_FILE)" \
        -H "Content-Type: application/json" \
        -d "{\"status\": \"$status\", \"log\": \"$log_msg\"}" > /dev/null
}
 
update_inventory() {
    local hostname=$(hostname)
    local os_info=$(uname -sr)
    local arch=$(uname -m)
    local free_disk=$(df -h / | tail -1 | awk '{print $4}')
 
    curl -s -X PATCH "$HARBOR_URL/api/v1/device/inventory" \
        -H "Authorization: Bearer $(cat $TOKEN_FILE)" \
        -H "Content-Type: application/json" \
        -d "{
            \"os\": \"$os_info\",
            \"arch\": \"$arch\",
            \"hostname\": \"$hostname\",
            \"free_disk\": \"$free_disk\"
        }" > /dev/null
}
 
check_deployments() {
    local TOKEN=$(cat "$TOKEN_FILE")
    RESPONSE=$(curl -s -w "\n%{http_code}" "$HARBOR_URL/api/v1/device/deployments/next" \
        -H "Authorization: Bearer $TOKEN")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
 
    [ "$HTTP_CODE" = "204" ] && return 0  # Nada pendente
    [ "$HTTP_CODE" != "200" ] && echo "[harbor] Erro: $HTTP_CODE" && return 1
 
    local DD_ID=$(echo "$BODY" | jq -r '.dd_id')
    local TARGET_PATH=$(echo "$BODY" | jq -r '.artifact.target_path')
    local FILE_MODE=$(echo "$BODY" | jq -r '.artifact.file_mode')
    local CHECKSUM=$(echo "$BODY" | jq -r '.artifact.checksum_sha256')
    local DOWNLOAD_URL=$(echo "$BODY" | jq -r '.artifact.download_url')
    local PRE_CMD=$(echo "$BODY" | jq -r '.artifact.pre_install_cmd // empty')
    local POST_CMD=$(echo "$BODY" | jq -r '.artifact.post_install_cmd // empty')
    local MAX_ATTEMPTS=$(echo "$BODY" | jq -r '.retry.max_attempts')
    local INTERVAL=$(echo "$BODY" | jq -r '.retry.interval_sec')
    local BACKOFF=$(echo "$BODY" | jq -r '.retry.backoff_multiplier')
    local NAME=$(echo "$BODY" | jq -r '.artifact.name')
    local VERSION=$(echo "$BODY" | jq -r '.artifact.version')
 
    echo "[harbor] Deployment recebido: $NAME v$VERSION -> $TARGET_PATH"
 
    # Download com retry e backoff exponencial
    report_status "$DD_ID" "downloading" "Iniciando download de $NAME v$VERSION"
    local ATTEMPT=0
    local WAIT=$INTERVAL
    local TMP_FILE="/tmp/harbor_download_$$"
 
    while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
        curl -s -o "$TMP_FILE" "$HARBOR_URL$DOWNLOAD_URL" \
            -H "Authorization: Bearer $TOKEN"
        local ACTUAL=$(sha256sum "$TMP_FILE" | cut -d' ' -f1)
 
        if [ "$ACTUAL" = "$CHECKSUM" ]; then
            break
        fi
 
        ATTEMPT=$((ATTEMPT + 1))
        echo "[harbor] Checksum falhou (tentativa $ATTEMPT/$MAX_ATTEMPTS), retry em ${WAIT}s"
        sleep $WAIT
        WAIT=$((WAIT * BACKOFF))
    done
 
    if [ "$ACTUAL" != "$CHECKSUM" ]; then
        report_status "$DD_ID" "failure" "Checksum invalido apos $MAX_ATTEMPTS tentativas"
        rm -f "$TMP_FILE"
        return 1
    fi
 
    # Instalar
    report_status "$DD_ID" "installing" "Checksum OK. Instalando em $TARGET_PATH"
 
    [ -n "$PRE_CMD" ] && echo "[harbor] pre_install: $PRE_CMD" && eval "$PRE_CMD"
 
    mkdir -p "$(dirname "$TARGET_PATH")"
    cp "$TMP_FILE" "$TARGET_PATH"
    chmod "$FILE_MODE" "$TARGET_PATH"
    rm -f "$TMP_FILE"
 
    [ -n "$POST_CMD" ] && echo "[harbor] post_install: $POST_CMD" && eval "$POST_CMD"
 
    report_status "$DD_ID" "success" "Instalado $NAME v$VERSION em $TARGET_PATH"
    echo "[harbor] Deploy concluido: $NAME v$VERSION"
}
 
# --- Main ---
mkdir -p /etc/harbor
echo "[harbor] Agent iniciado (servidor: $HARBOR_URL, poll: ${POLL_INTERVAL}s)"
 
INVENTORY_COUNTER=0
while true; do
    # Autenticar se necessario
    if [ ! -f "$TOKEN_FILE" ] || [ ! -s "$TOKEN_FILE" ]; then
        authenticate || { sleep $POLL_INTERVAL; continue; }
    fi
 
    # Atualizar inventory a cada 10 ciclos
    INVENTORY_COUNTER=$((INVENTORY_COUNTER + 1))
    if [ $INVENTORY_COUNTER -ge 10 ]; then
        update_inventory
        INVENTORY_COUNTER=0
    fi
 
    # Verificar deployments
    check_deployments
 
    sleep $POLL_INTERVAL
done
```
 
### Instalando o Agent no Device
 
```bash
# 1. Copie o script para o device
scp harbor-agent.sh pi@192.168.1.50:/usr/local/bin/
ssh pi@192.168.1.50 "chmod +x /usr/local/bin/harbor-agent.sh"
 
# 2. Crie o servico systemd
ssh pi@192.168.1.50 "cat > /etc/systemd/system/harbor-agent.service" << 'EOF'
[Unit]
Description=Harbor Agent
After=network-online.target
Wants=network-online.target
 
[Service]
Type=simple
Environment=HARBOR_URL=http://harbor-server:8080
Environment=DEVICE_TYPE=raspberry-pi-4
Environment=POLL_INTERVAL=60
ExecStart=/usr/local/bin/harbor-agent.sh
Restart=always
RestartSec=10
 
[Install]
WantedBy=multi-user.target
EOF
 
# 3. Ative e inicie o servico
ssh pi@192.168.1.50 "systemctl daemon-reload && systemctl enable --now harbor-agent"
 
# 4. Verifique os logs
ssh pi@192.168.1.50 "journalctl -u harbor-agent -f"
```
 
O device aparecera como `pending` no Harbor. Aceite-o pela Management API e ele comecara a receber deployments.
 
---
 
## Fluxo Completo
 
Exemplo passo a passo: deploy de uma nova versao de `myapp` em todos os Raspberry Pi de producao.
 
```bash
# 1. Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/management/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@harbor.local","password":"admin"}' | jq -r '.token')
 
# 2. Upload do artifact
ARTIFACT=$(curl -s -X POST http://localhost:8080/api/v1/management/artifacts \
  -H "Authorization: Bearer $TOKEN" \
  -F "name=myapp" \
  -F "version=2.0.0" \
  -F "target_path=/usr/local/bin/myapp" \
  -F "file_mode=0755" \
  -F "device_types=raspberry-pi-4" \
  -F "pre_install_cmd=systemctl stop myapp" \
  -F "post_install_cmd=systemctl start myapp" \
  -F "file=@./build/myapp-arm64")
ARTIFACT_ID=$(echo $ARTIFACT | jq -r '.id')
echo "Artifact criado: $ARTIFACT_ID"
 
# 3. Criar deployment para todos os RPi de producao
DEPLOYMENT=$(curl -s -X POST http://localhost:8080/api/v1/management/deployments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"deploy-myapp-v2.0.0\",
    \"artifact_id\": \"$ARTIFACT_ID\",
    \"target_device_tags\": [\"production\"],
    \"target_device_types\": [\"raspberry-pi-4\"]
  }")
DEPLOY_ID=$(echo $DEPLOYMENT | jq -r '.id')
echo "Deployment criado: $DEPLOY_ID"
 
# 4. Acompanhar progresso
watch -n 5 "curl -s http://localhost:8080/api/v1/management/deployments/$DEPLOY_ID/devices \
  -H 'Authorization: Bearer $TOKEN' | jq '.data[] | {device_id, status, log}'"
```
 
---
 
## Variaveis de Ambiente
 
| Variavel                      | Default                     | Descricao                          |
|-------------------------------|-----------------------------|------------------------------------|
| `HARBOR_HOST`                 | `0.0.0.0`                  | Endereco de bind do servidor       |
| `HARBOR_PORT`                 | `8080`                     | Porta HTTP                         |
| `HARBOR_DB_HOST`              | `localhost`                | Host do PostgreSQL                 |
| `HARBOR_DB_PORT`              | `5432`                     | Porta do PostgreSQL                |
| `HARBOR_DB_NAME`              | `harbor`                   | Nome do banco                      |
| `HARBOR_DB_USER`              | `harbor`                   | Usuario do banco                   |
| `HARBOR_DB_PASSWORD`          | `harbor`                   | Senha do banco                     |
| `HARBOR_DB_SSLMODE`           | `disable`                  | Modo SSL do PostgreSQL             |
| `HARBOR_JWT_SECRET`           | `change-me-in-production`  | Chave para assinar JWTs            |
| `HARBOR_JWT_EXPIRY`           | `24h`                      | Validade do JWT                    |
| `HARBOR_DEVICE_TOKEN_EXPIRY`  | `8760h` (1 ano)            | Validade do token de device        |
| `HARBOR_STORAGE_PATH`         | `/data/artifacts`          | Diretorio de armazenamento         |
| `HARBOR_CORS_ORIGINS`         | `http://localhost:3000`    | Origens CORS (separadas por `,`)   |
 
---
 
## Endpoints
 
### Operacionais
 
| Metodo | Endpoint   | Descricao                |
|--------|-----------|--------------------------|
| GET    | `/health`  | Health check             |
| GET    | `/metrics` | Metricas Prometheus      |
| GET    | `/docs`    | Swagger UI da API        |
| GET    | `/docs/openapi.yaml` | Spec OpenAPI 3.0 |
 
### Device API (`/api/v1/device`)
 
| Metodo | Endpoint                     | Auth   | Descricao                        |
|--------|------------------------------|--------|----------------------------------|
| POST   | `/auth`                      | Nao    | Registrar/autenticar device      |
| GET    | `/deployments/next`          | Token  | Buscar proximo deployment        |
| PUT    | `/deployments/{id}/status`   | Token  | Reportar status                  |
| GET    | `/deployments/{id}/download` | Token  | Download do artifact             |
| PATCH  | `/inventory`                 | Token  | Atualizar inventory              |
 
### Management API (`/api/v1/management`)
 
| Metodo | Endpoint                       | Auth | Descricao                    |
|--------|--------------------------------|------|------------------------------|
| POST   | `/auth/login`                  | Nao  | Login (email/senha -> JWT)   |
| POST   | `/auth/refresh`                | JWT  | Renovar JWT                  |
| GET    | `/devices`                     | JWT  | Listar devices               |
| GET    | `/devices/count`               | JWT  | Contagem por status          |
| GET    | `/devices/{id}`                | JWT  | Detalhes do device           |
| PUT    | `/devices/{id}/status`         | JWT  | Aceitar/rejeitar device      |
| PATCH  | `/devices/{id}/tags`           | JWT  | Atualizar tags               |
| DELETE | `/devices/{id}`                | JWT  | Decommission                 |
| GET    | `/artifacts`                   | JWT  | Listar artifacts             |
| POST   | `/artifacts`                   | JWT  | Upload (multipart)           |
| GET    | `/artifacts/{id}`              | JWT  | Detalhes do artifact         |
| GET    | `/artifacts/{id}/download`     | JWT  | Download do arquivo          |
| DELETE | `/artifacts/{id}`              | JWT  | Remover artifact             |
| GET    | `/deployments`                 | JWT  | Listar deployments           |
| POST   | `/deployments`                 | JWT  | Criar deployment             |
| GET    | `/deployments/statistics`      | JWT  | Estatisticas                 |
| GET    | `/deployments/{id}`            | JWT  | Detalhes do deployment       |
| POST   | `/deployments/{id}/cancel`     | JWT  | Cancelar deployment          |
| GET    | `/deployments/{id}/devices`    | JWT  | Status por device            |
| GET    | `/audit`                       | JWT  | Log de auditoria             |
 
---
 
## Desenvolvimento

```bash
# Build
make build
 
# Rodar testes
make test
 
# Lint
make lint
 
# Docker
make docker-up     # Subir
make docker-down   # Parar
make docker-logs   # Ver logs
```

### Dashboard Frontend (WIP)

O frontend inicial do dashboard esta em `frontend/dashboard`.

```bash
cd frontend/dashboard
cp .env.example .env
npm install
npm run dev
```

O app sobe em `http://localhost:5173` e usa proxy para a API em `http://localhost:8080`.

### Estrutura do Projeto

```
Harbor/
├── cmd/harbor/main.go              # Entrypoint
├── internal/
│   ├── api/                        # Camada HTTP (handlers, middleware, router)
│   │   ├── device/                 # Endpoints para devices (agent)
│   │   ├── management/             # Endpoints para frontend (React)
│   │   ├── middleware/             # Auth, logging, rate limit, audit, metrics
│   │   └── response/              # Helpers de resposta JSON
│   ├── auth/                       # JWT e tokens de device
│   ├── config/                     # Configuracao via env vars
│   ├── domain/                     # Entidades e interfaces
│   ├── repository/postgres/        # Implementacao PostgreSQL
│   ├── service/                    # Logica de negocio
│   └── storage/                    # Armazenamento de arquivos
├── migrations/                     # SQL migrations
├── frontend/dashboard/             # Dashboard React (WIP)
├── Dockerfile                      # Build multi-stage
├── docker-compose.yml              # Dev environment
└── Makefile                        # Comandos uteis
```
