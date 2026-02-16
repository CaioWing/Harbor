# Harbor — Plano de Projeto

> Sistema de deploy de arquivos individuais (single-file OTA) para dispositivos remotos.
> Substituto do Mender focado em simplicidade, eficiência e granularidade por arquivo.

---

## 1. Visao Geral

Harbor e um sistema de gerenciamento de deployments que permite atualizar **arquivos individuais** em dispositivos remotos de forma controlada. Diferente do Mender (que trabalha com imagens de rootfs completas), o Harbor opera no nivel de arquivo, permitindo:

- Atualizar um unico binario, config ou script sem re-flashar o sistema inteiro
- Deployments mais rapidos e com menor uso de banda
- Rollback granular por arquivo
- Gerenciamento centralizado via API REST (consumida pelo frontend React)

### Analogia com Mender

| Conceito Mender         | Equivalente Harbor              |
|--------------------------|----------------------------------|
| Artifact (.mender)       | Artifact (arquivo unico + meta)  |
| Device Identity          | Device (fingerprint + atributos) |
| Deployment               | Deployment (artifact -> devices) |
| Device Groups            | Device Groups (tags/labels)      |
| rootfs-image update      | Single-file update               |
| mender-client            | harbor-agent (futuro)            |

---

## 2. Identidade de Dispositivos

### 2.1 Device Identity (Fingerprint)

Cada dispositivo tem uma identidade unica composta por:

```
Device Identity = {
    "mac_address":    "aa:bb:cc:dd:ee:ff",     // obrigatorio
    "serial_number":  "SN-2024-001",            // opcional
    "device_type":    "raspberry-pi-4",          // obrigatorio
    "custom_id":      "planta-sp-rack-03"        // opcional, definido pelo usuario
}
```

- **Identity Hash**: SHA-256 dos campos de identidade → gera um `device_id` deterministico
- O dispositivo envia sua identidade no primeiro contato (admission request)
- O operador aceita/rejeita o dispositivo via API de gerenciamento
- Apos aceito, o dispositivo recebe um **auth token** (JWT) para comunicacao futura

### 2.2 Device Inventory (Atributos)

Alem da identidade, cada dispositivo reporta atributos dinamicos:

```
Inventory = {
    "os":             "Linux 5.15",
    "arch":           "arm64",
    "hostname":       "edge-node-03",
    "ip_address":     "192.168.1.50",
    "harbor_version": "0.1.0",
    "free_disk":      "2.3GB",
    "custom_attrs":   { "location": "SP", "environment": "production" }
}
```

- Inventory e atualizado periodicamente pelo agent
- Usado para filtrar dispositivos na hora de criar deployments
- Indexado no banco para queries eficientes

### 2.3 Device States

```
[new] --> PENDING --> ACCEPTED --> ACTIVE
                  \-> REJECTED

ACTIVE --> DECOMMISSIONED  (removido permanentemente)
```

---

## 3. Arquitetura do Backend

### 3.1 Estrutura de Diretorios

```
Harbor/
├── cmd/
│   └── harbor/
│       └── main.go                  # Entrypoint
│
├── internal/
│   ├── config/
│   │   └── config.go                # Configuracao via env vars
│   │
│   ├── domain/                      # Entidades e interfaces (camada pura)
│   │   ├── device.go                # Device, DeviceIdentity, DeviceInventory
│   │   ├── artifact.go              # Artifact, ArtifactMeta
│   │   ├── deployment.go            # Deployment, DeploymentDevice
│   │   └── errors.go                # Erros de dominio
│   │
│   ├── service/                     # Logica de negocio
│   │   ├── device_service.go        # Admissao, auth, inventory
│   │   ├── artifact_service.go      # Upload, validacao, checksum
│   │   └── deployment_service.go    # Criacao, orquestracao, status
│   │
│   ├── repository/                  # Implementacoes de persistencia
│   │   └── postgres/
│   │       ├── device_repo.go
│   │       ├── artifact_repo.go
│   │       ├── deployment_repo.go
│   │       └── migrations.go        # Embedded migrations
│   │
│   ├── storage/                     # Armazenamento de arquivos
│   │   ├── store.go                 # Interface FileStore
│   │   └── local/
│   │       └── local.go             # Implementacao filesystem local
│   │
│   ├── api/                         # Camada HTTP
│   │   ├── router.go                # Setup do chi router
│   │   ├── middleware/
│   │   │   ├── auth.go              # JWT validation
│   │   │   ├── device_auth.go       # Device token validation
│   │   │   └── logging.go           # Request logging (slog)
│   │   ├── management/              # Endpoints de gerenciamento (frontend)
│   │   │   ├── device_handler.go
│   │   │   ├── artifact_handler.go
│   │   │   └── deployment_handler.go
│   │   └── device/                  # Endpoints para dispositivos (agent)
│   │       ├── auth_handler.go
│   │       ├── deployment_handler.go
│   │       └── inventory_handler.go
│   │
│   └── auth/                        # Autenticacao
│       ├── jwt.go                   # Geracao/validacao JWT (gerenciamento)
│       └── device_token.go          # Geracao/validacao token de device
│
├── migrations/
│   ├── 001_devices.up.sql
│   ├── 001_devices.down.sql
│   ├── 002_artifacts.up.sql
│   ├── 002_artifacts.down.sql
│   ├── 003_deployments.up.sql
│   └── 003_deployments.down.sql
│
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── PLAN.md
```

### 3.2 Stack Tecnica

| Componente        | Tecnologia                                      |
|-------------------|--------------------------------------------------|
| Linguagem         | Go 1.22+                                         |
| HTTP Router       | chi (leve, stdlib-compatible)                    |
| Banco de Dados    | PostgreSQL 16                                     |
| Driver DB         | pgx/v5                                            |
| Migrations        | golang-migrate                                    |
| Auth              | JWT (golang-jwt/jwt/v5)                           |
| Logging           | slog (stdlib)                                     |
| Config            | envconfig ou env vars direto                      |
| File Storage      | Filesystem local (abstrato via interface)         |
| Containerizacao   | Docker + Docker Compose                           |
| Testes            | testing (stdlib) + testcontainers-go              |

### 3.3 Dependencias (go.mod)

```
github.com/go-chi/chi/v5          # Router
github.com/jackc/pgx/v5           # PostgreSQL driver
github.com/golang-migrate/migrate  # DB migrations
github.com/golang-jwt/jwt/v5      # JWT tokens
github.com/google/uuid             # UUIDs
golang.org/x/crypto                # Hashing
```

---

## 4. Database Schema

### 4.1 devices

```sql
CREATE TABLE devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity_hash   VARCHAR(64) UNIQUE NOT NULL,  -- SHA-256 da identidade
    identity_data   JSONB NOT NULL,               -- mac, serial, device_type, etc
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- enum: pending, accepted, rejected, decommissioned

    auth_token_hash VARCHAR(64),                  -- hash do token atual

    inventory       JSONB DEFAULT '{}',           -- atributos dinamicos
    device_type     VARCHAR(100) NOT NULL,        -- extraido de identity_data

    tags            TEXT[] DEFAULT '{}',           -- labels para agrupamento

    last_check_in   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_devices_status ON devices(status);
CREATE INDEX idx_devices_device_type ON devices(device_type);
CREATE INDEX idx_devices_tags ON devices USING GIN(tags);
CREATE INDEX idx_devices_inventory ON devices USING GIN(inventory);
```

### 4.2 artifacts

```sql
CREATE TABLE artifacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    version         VARCHAR(100) NOT NULL,
    description     TEXT DEFAULT '',

    -- Metadados do arquivo
    file_name       VARCHAR(255) NOT NULL,        -- nome original do arquivo
    file_size       BIGINT NOT NULL,
    checksum_sha256 VARCHAR(64) NOT NULL,

    -- Onde o arquivo deve ser colocado no device
    target_path     VARCHAR(500) NOT NULL,         -- ex: /usr/local/bin/myapp

    -- Permissoes do arquivo no device (opcionais)
    file_mode       VARCHAR(10) DEFAULT '0644',    -- ex: 0755
    file_owner      VARCHAR(100) DEFAULT '',       -- ex: root:root

    -- Compatibilidade
    device_types    TEXT[] NOT NULL,                -- tipos de device compativeis

    -- Storage
    storage_path    VARCHAR(500) NOT NULL,          -- caminho no storage local

    -- Hooks (comandos para executar no device)
    pre_install_cmd  TEXT DEFAULT '',               -- comando antes de instalar
    post_install_cmd TEXT DEFAULT '',               -- comando depois de instalar
    rollback_cmd     TEXT DEFAULT '',               -- comando em caso de falha

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(name, version)
);

CREATE INDEX idx_artifacts_device_types ON artifacts USING GIN(device_types);
```

### 4.3 deployments

```sql
CREATE TABLE deployments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    artifact_id     UUID NOT NULL REFERENCES artifacts(id),

    status          VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    -- enum: scheduled, active, completed, cancelled

    -- Filtros de destino
    target_device_ids   UUID[] DEFAULT NULL,       -- devices especificos OU
    target_device_tags  TEXT[] DEFAULT NULL,        -- devices por tags
    target_device_types TEXT[] DEFAULT NULL,        -- devices por tipo

    max_parallel    INT DEFAULT 0,                  -- 0 = sem limite

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ
);
```

### 4.4 deployment_devices

```sql
CREATE TABLE deployment_devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id   UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    device_id       UUID NOT NULL REFERENCES devices(id),

    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- enum: pending, downloading, installing, success, failure, skipped

    attempts        INT DEFAULT 0,
    log             TEXT DEFAULT '',                 -- output do device

    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,

    UNIQUE(deployment_id, device_id)
);

CREATE INDEX idx_dd_deployment ON deployment_devices(deployment_id);
CREATE INDEX idx_dd_device ON deployment_devices(device_id);
CREATE INDEX idx_dd_status ON deployment_devices(status);
```

---

## 5. API Design

### 5.1 Device API (chamada pelo harbor-agent nos dispositivos)

Base path: `/api/v1/device`

| Metodo | Endpoint                          | Descricao                                          |
|--------|-----------------------------------|----------------------------------------------------|
| POST   | /auth                             | Enviar identidade, receber auth token               |
| GET    | /deployments/next                 | Buscar proximo deployment pendente                  |
| PUT    | /deployments/{id}/status          | Reportar status do deployment (downloading, etc)   |
| PATCH  | /inventory                        | Atualizar inventory/atributos do device             |

#### POST /api/v1/device/auth

Request:
```json
{
    "identity": {
        "mac_address": "aa:bb:cc:dd:ee:ff",
        "device_type": "raspberry-pi-4",
        "serial_number": "SN-001"
    }
}
```

Response (device aceito):
```json
{
    "token": "eyJhbG...",
    "expires_at": "2026-03-16T00:00:00Z",
    "server_url": "https://harbor.example.com"
}
```

Response (device pendente): `401 Unauthorized`

#### GET /api/v1/device/deployments/next

Headers: `Authorization: Bearer <device_token>`

Response (deployment disponivel):
```json
{
    "deployment_id": "uuid",
    "artifact": {
        "name": "myapp",
        "version": "1.2.0",
        "target_path": "/usr/local/bin/myapp",
        "file_mode": "0755",
        "checksum_sha256": "abc123...",
        "file_size": 1048576,
        "download_url": "/api/v1/device/deployments/{id}/download",
        "pre_install_cmd": "systemctl stop myapp",
        "post_install_cmd": "systemctl start myapp"
    }
}
```

Response (nada pendente): `204 No Content`

#### PUT /api/v1/device/deployments/{id}/status

```json
{
    "status": "success",
    "log": "Downloaded OK. Checksum verified. Service restarted."
}
```

### 5.2 Management API (chamada pelo frontend React)

Base path: `/api/v1/management`

#### Devices

| Metodo | Endpoint                  | Descricao                          |
|--------|---------------------------|------------------------------------|
| GET    | /devices                  | Listar devices (com filtros)       |
| GET    | /devices/{id}             | Detalhes de um device              |
| PUT    | /devices/{id}/status      | Aceitar/rejeitar device            |
| PATCH  | /devices/{id}/tags        | Adicionar/remover tags             |
| DELETE | /devices/{id}             | Decommission device                |
| GET    | /devices/count            | Contagem por status                |

#### Artifacts

| Metodo | Endpoint                      | Descricao                      |
|--------|-------------------------------|--------------------------------|
| GET    | /artifacts                    | Listar artifacts               |
| POST   | /artifacts                    | Upload de novo artifact        |
| GET    | /artifacts/{id}               | Detalhes do artifact           |
| GET    | /artifacts/{id}/download      | Download do arquivo            |
| DELETE | /artifacts/{id}               | Remover artifact               |

#### Deployments

| Metodo | Endpoint                          | Descricao                           |
|--------|-----------------------------------|-------------------------------------|
| GET    | /deployments                      | Listar deployments                  |
| POST   | /deployments                      | Criar novo deployment               |
| GET    | /deployments/{id}                 | Detalhes + status por device        |
| POST   | /deployments/{id}/cancel          | Cancelar deployment                 |
| GET    | /deployments/{id}/devices         | Status de cada device no deployment |
| GET    | /deployments/statistics           | Estatisticas gerais                 |

#### Auth (Management)

| Metodo | Endpoint              | Descricao                          |
|--------|-----------------------|------------------------------------|
| POST   | /auth/login           | Login com email/senha → JWT        |
| POST   | /auth/refresh         | Renovar JWT                        |

### 5.3 Consideracoes para o Frontend React

- Todos os endpoints retornam JSON
- Listagens suportam paginacao: `?page=1&per_page=20`
- Listagens suportam ordenacao: `?sort=created_at&order=desc`
- Filtros via query params: `?status=accepted&device_type=raspberry-pi-4&tag=production`
- CORS configuravel via env var
- Respostas paginadas seguem formato:
```json
{
    "data": [...],
    "pagination": {
        "page": 1,
        "per_page": 20,
        "total": 150,
        "total_pages": 8
    }
}
```

---

## 6. Fluxos Principais

### 6.1 Admissao de Device

```
Device                          Harbor Server                    Operador (React)
  |                                   |                               |
  |-- POST /device/auth ------------->|                               |
  |   {identity: {...}}               |                               |
  |                                   |-- Cria device "pending" ----->|
  |<---- 401 Unauthorized ------------|                               |
  |                                   |                               |
  |                                   |<-- PUT /devices/{id}/status --|
  |                                   |    {status: "accepted"}       |
  |                                   |                               |
  |-- POST /device/auth ------------->|                               |
  |   {identity: {...}}               |                               |
  |<---- 200 {token: "..."}  --------|                               |
  |                                   |                               |
  |== Device agora autenticado =======================================|
```

### 6.2 Deploy de Arquivo

```
Operador (React)                Harbor Server                    Device
  |                                   |                               |
  |-- POST /artifacts -------------->|                               |
  |   (multipart: file + metadata)    |-- Salva arquivo no storage    |
  |<-- 201 {artifact_id} ------------|                               |
  |                                   |                               |
  |-- POST /deployments ------------>|                               |
  |   {artifact_id, targets}          |-- Resolve targets → devices   |
  |<-- 201 {deployment_id} ----------|-- Cria deployment_devices      |
  |                                   |                               |
  |                                   |   (device faz polling)        |
  |                                   |<-- GET /deployments/next -----|
  |                                   |--- 200 {artifact info} ------>|
  |                                   |                               |
  |                                   |<-- GET /download -------------|
  |                                   |--- file stream -------------->|
  |                                   |                               |-- Verifica checksum
  |                                   |                               |-- Executa pre_install
  |                                   |                               |-- Substitui arquivo
  |                                   |                               |-- Executa post_install
  |                                   |                               |
  |                                   |<-- PUT /status {success} -----|
  |                                   |-- Atualiza deployment_device   |
  |                                   |                               |
  |-- GET /deployments/{id} -------->|                               |
  |<-- {status: completed} ----------|                               |
```

### 6.3 Rollback

```
Operador (React)                Harbor Server                    Device
  |                                   |                               |
  |-- POST /deployments ------------>|                               |
  |   {artifact_id: <versao_anterior>}|                               |
  |   (mesmo target_path, versao old) |-- Cria novo deployment        |
  |                                   |                               |
  |   ... mesmo fluxo de deploy ...                                   |
```

Rollback = criar um novo deployment apontando para o artifact da versao anterior.
O Harbor mantem historico de todos os artifacts, entao qualquer versao pode ser re-deployada.

---

## 7. MVP — Features e Prioridades

### Fase 1: Core (Semanas 1-2) — CONCLUIDA
> Fundacao: dispositivos se conectam, arquivos sao enviados.

- [x] Setup do projeto (go.mod, estrutura de dirs, docker-compose)
- [x] **Config**: Leitura de env vars, struct de configuracao
- [x] **Database**: Conexao PostgreSQL + migrations (pgx + golang-migrate)
- [x] **Device Identity**: Registro, fingerprint SHA-256, estados (pending/accepted/rejected/decommissioned)
- [x] **Device Auth**: Geracao de token opaco no aceite, validacao via middleware
- [x] **Artifact Upload**: Upload multipart (500MB limit), checksum SHA-256 via TeeReader, storage local
- [x] **Artifact Metadata**: CRUD completo (list, get, upload, download, delete)
- [x] Testes unitarios dos services (42 testes: device, artifact, deployment)

### Fase 2: Deployments (Semanas 3-4) — CONCLUIDA
> Funcionalidade principal: orquestrar updates nos devices.

- [x] **Criar Deployment**: Selecionar artifact + devices/tags destino
- [x] **Resolver Targets**: Expansao de device IDs, tags e device types em lista de devices
- [x] **Device Polling**: Endpoint GET /deployments/next com detalhes do artifact
- [x] **Download de Artifact**: Stream do arquivo com header X-Checksum-SHA256
- [x] **Status Tracking**: Device reporta progresso (downloading/installing/success/failure)
- [x] **Deployment Lifecycle**: scheduled → active → completed/cancelled com cancel endpoint
- [ ] Testes de integracao com testcontainers

### Fase 3: Gerenciamento (Semanas 5-6) — CONCLUIDA
> API completa para o frontend React consumir.

- [x] **Management Auth**: Login com email/senha → JWT, middleware de validacao
- [x] **Device Listing**: Filtros (status, device_type, tags), paginacao, ordenacao
- [x] **Device Tags**: Adicionar/remover tags via PATCH endpoint
- [x] **Device Inventory**: Atributos reportados via PATCH /inventory pelo agent
- [x] **Deployment Dashboard**: Listagem com paginacao + endpoint /statistics (total, por status)
- [x] **Deployment Details**: Status por device via GET /deployments/{id}/devices
- [x] **Artifact Management**: Listagem, download, remocao com cleanup do storage
- [x] **CORS**: Configuravel via HARBOR_CORS_ORIGINS, expondo headers necessarios
- [ ] Testes end-to-end

### Fase 4: Robustez (Semana 7+) — CONCLUIDA
> Producao-ready.

- [x] **Retry Logic**: Configuracao de retry enviada ao device (max_attempts, interval, backoff)
- [x] **Checksum Verification**: SHA-256 calculado no upload e enviado via header no download
- [x] **Rate Limiting**: Per-IP token bucket (device: 10req/s, management: 30req/s) com cleanup automatico
- [x] **Audit Log**: Tabela audit_log + middleware automatico para acoes de management + endpoint GET /audit
- [x] **Graceful Shutdown**: Encerramento limpo com signal handling (SIGINT/SIGTERM) e timeout de 10s
- [x] **Health Check**: Endpoint GET /health retorna {"status": "ok"}
- [x] **Metricas**: Prometheus-compatible GET /metrics (requests total, duration, active requests)
- [x] **Cleanup Job**: Scheduler periodico (6h) remove artifacts orfaos do storage
- [x] **Auth Refresh**: POST /auth/refresh para renovar JWT sem re-login

---

## 8. Comparacao: O que o Harbor substitui do Mender

| Feature Mender                | Harbor MVP        | Harbor Futuro       |
|-------------------------------|-------------------|---------------------|
| Device admission              | Sim               | Sim                 |
| Device inventory              | Sim               | Sim                 |
| Device groups                 | Tags              | Tags + filtros      |
| Artifact management           | Sim (single-file) | Sim                 |
| Deployments                   | Sim               | Sim                 |
| Deployment status/logs        | Sim               | Sim                 |
| Rollback                      | Via re-deploy     | Via re-deploy       |
| rootfs updates                | Nao (by design)   | Nao                 |
| Delta updates                 | Nao               | Possivel (bsdiff)   |
| mender-client (agent)         | Nao (API only)    | harbor-agent (Go)   |
| Dashboard UI                  | Nao (API only)    | Frontend React      |
| mTLS                          | Nao               | Possivel            |
| Multi-tenancy                 | Nao               | Possivel            |

---

## 9. Decisoes Tecnicas

### 9.1 Por que polling (e nao push)?
- Dispositivos em redes restritas (NAT, firewall) nao aceitam conexoes de entrada
- Polling e mais simples e confiavel
- Intervalo configuravel (default: 60s)
- Futuro: WebSocket para notificacao instantanea (opcional)

### 9.2 Por que SHA-256 para identity hash?
- Deterministico: mesmos campos → mesmo hash → mesmo device
- Evita duplicatas mesmo se o device reinstalar o agent
- Seguro contra colisoes

### 9.3 Por que JWT para management e token opaco para devices?
- **Management JWT**: Stateless, frontend armazena no localStorage, expira em 24h
- **Device token**: Opaco (random + hash no banco), vida longa, revogavel individualmente

### 9.4 Estrategia de rollback
- Nao ha "rollback automatico" no MVP
- Rollback = operador cria novo deployment com artifact de versao anterior
- Simples, explicito, auditavel
- Futuro: rollback automatico se post_install_cmd falhar

---

## 10. Variaveis de Ambiente

```bash
# Server
HARBOR_PORT=8080
HARBOR_HOST=0.0.0.0

# Database
HARBOR_DB_HOST=localhost
HARBOR_DB_PORT=5432
HARBOR_DB_NAME=harbor
HARBOR_DB_USER=harbor
HARBOR_DB_PASSWORD=harbor
HARBOR_DB_SSLMODE=disable

# Auth
HARBOR_JWT_SECRET=change-me-in-production
HARBOR_JWT_EXPIRY=24h
HARBOR_DEVICE_TOKEN_EXPIRY=8760h  # 1 ano

# Storage
HARBOR_STORAGE_PATH=/data/artifacts

# CORS (para frontend React)
HARBOR_CORS_ORIGINS=http://localhost:3000,https://harbor.example.com

# Polling
HARBOR_DEVICE_POLL_INTERVAL=60s
```

---

## 11. Docker Compose (Dev Environment)

```yaml
services:
  harbor:
    build: .
    ports:
      - "8080:8080"
    environment:
      - HARBOR_DB_HOST=postgres
      - HARBOR_DB_PORT=5432
      - HARBOR_DB_NAME=harbor
      - HARBOR_DB_USER=harbor
      - HARBOR_DB_PASSWORD=harbor
      - HARBOR_JWT_SECRET=dev-secret
      - HARBOR_STORAGE_PATH=/data/artifacts
      - HARBOR_CORS_ORIGINS=http://localhost:3000
    volumes:
      - artifacts:/data/artifacts
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: harbor
      POSTGRES_USER: harbor
      POSTGRES_PASSWORD: harbor
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U harbor"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
  artifacts:
```

---

## 12. Progresso da Implementacao

### Concluido

1. ~~**Inicializar projeto Go**~~ — go.mod, dependencias, estrutura de dirs
2. ~~**Config**~~ — Struct de configuracao com env vars e defaults
3. ~~**Database**~~ — Conexao pgx pool + embedded migrations (golang-migrate)
4. ~~**Domain**~~ — Entidades Device, Artifact, Deployment com interfaces de repositorio
5. ~~**Repository**~~ — Implementacao PostgreSQL completa (devices, artifacts, deployments)
6. ~~**Storage**~~ — FileStore local com diretorio por data (YYYY/MM/DD)
7. ~~**Device Service**~~ — Admissao, auth token opaco, SHA-256 identity hash
8. ~~**Device API**~~ — POST /auth, PATCH /inventory, GET /deployments/next, download, status
9. ~~**Artifact Service**~~ — Upload multipart, validacao, SHA-256 checksum
10. ~~**Artifact API**~~ — POST/GET/DELETE + download com checksum header
11. ~~**Deployment Service**~~ — Criacao, resolucao de targets (IDs/tags/tipos), lifecycle
12. ~~**Deployment API**~~ — CRUD + cancel + status por device + statistics
13. ~~**Device Polling**~~ — GET /deployments/next + stream download + status report
14. ~~**Management Auth**~~ — JWT login com bcrypt password hash
15. ~~**Management API**~~ — Todos os endpoints com paginacao, filtros e ordenacao
16. ~~**Docker**~~ — Dockerfile multi-stage + docker-compose.yml com healthcheck
17. ~~**CORS**~~ — Configuravel via env var, headers expostos
18. ~~**Health Check**~~ — GET /health endpoint
19. ~~**Graceful Shutdown**~~ — Signal handling + timeout

### Concluido (Fase 4)

20. ~~**Unit Tests**~~ — 42 testes cobrindo DeviceService, ArtifactService, DeploymentService com mocks
21. ~~**Rate Limiting**~~ — Per-IP token bucket middleware (device 10req/s, management 30req/s)
22. ~~**Audit Log**~~ — Tabela audit_log, AuditService, middleware automatico, endpoint GET /audit
23. ~~**Cleanup Job**~~ — Scheduler a cada 6h remove artifacts orfaos (sem deployment ativo + storage missing)
24. ~~**Auth Refresh**~~ — POST /auth/refresh gera novo JWT para usuario autenticado
25. ~~**Retry Logic**~~ — Configuracao de retry (max_attempts, interval, backoff) enviada ao device
26. ~~**Metricas**~~ — Prometheus GET /metrics (requests total por method/status, duration, active requests)

### Pendente

1. **Testes de integracao** — Testcontainers-go com PostgreSQL real
2. **Documentacao** — OpenAPI/Swagger spec para o frontend
3. **Testes end-to-end** — Fluxo completo device → deploy → status
