# Harbor Frontend (WIP)

Frontend React para o painel de operacao do Harbor.

## Stack

- React 18
- Vite
- React Router DOM

## Estrutura

- `src/pages`: telas e rotas
- `src/components`: componentes reutilizaveis
- `src/services`: integracao HTTP com API
- `src/context`: estado global de autenticacao
- `src/hooks`: hooks reutilizaveis

## Funcionalidades implementadas (WIP)

- Dashboard com metricas, graficos e auto-refresh configuravel
- Listagens de devices, deployments e auditoria com filtros + paginacao
- Sincronizacao de filtros/pagina via query params na URL
- Tela de detalhe de device com acoes de aceitar/rejeitar
- Tela de detalhe de deployment e listagem com acao de cancelar deployment

## Rodando localmente

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

A aplicacao sobe em `http://localhost:5173` e faz proxy de `/api` para `http://localhost:8080`.
