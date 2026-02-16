# Harbor Dashboard (WIP)

Frontend inicial do painel de operacao do Harbor.

## Funcionalidades iniciais

- Login com Management API (`/api/v1/management/auth/login`)
- Cards de status de devices e deployments
- Tabelas com ultimos deployments e devices
- Timeline com auditoria recente

## Rodando localmente

```bash
cd frontend/dashboard
cp .env.example .env
npm install
npm run dev
```

Por padrao o Vite sobe em `http://localhost:5173` e faz proxy de `/api` para `http://localhost:8080`.
