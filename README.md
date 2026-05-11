# HomePlan

HomePlan is evolving from a static GitHub Pages dashboard into a fullstack home project tracker: Todoist plus a rough floor plan plus AI-ready project/material planning for homeowners.

This milestone keeps the existing dashboard behavior while adding:

- Vue 3 + TypeScript + Vite frontend in `web/`
- Thin Go REST API in `api/`
- Postgres relational shell plus JSONB house state migrations in `db/migrations/`
- Helm chart for frontend, API, and Postgres in `deploy/helm/homeplan/`
- Minikube runner for local integration testing

The original static dashboard files are preserved in `legacy_static/` as a reference snapshot.

## Local Minikube

Minikube is the preferred fullstack local workflow.

```powershell
.\scripts\run-minikube.ps1
```

Then visit `http://localhost:8080`.

Optional parameters:

```powershell
.\scripts\run-minikube.ps1 -Namespace homeplan -Release homeplan -Port 8080
.\scripts\run-minikube.ps1 -SkipBuild
```

The script verifies `minikube`, `kubectl`, and `helm`, starts Minikube if needed, builds the API and web images inside Minikube, deploys the Helm chart, waits for readiness, and starts a local port-forward.

## Frontend

```powershell
cd web
npm install
npm run typecheck
npm run build
npm run dev
```

The frontend tries `GET /api/house/current` first. If the API is unavailable or no anonymous house has been saved yet, it falls back to the bundled seed state converted from the original `tasks.json` dashboard.

## API

```powershell
cd api
go test ./...
go run ./cmd/homeplan-api
```

Endpoints:

- `GET /healthz`
- `GET /api/house/current`
- `PUT /api/house/current`

Anonymous sessions use an opaque `homeplan_session` cookie. Project data is stored server-side in Postgres, not in the cookie.

## Database

The first migration creates:

- `users`
- `anonymous_sessions`
- `houses`
- `house_members`
- `house_state`
- `house_versions`
- `house_events`
- `proposed_changes`
- `ai_runs`
- `api_tokens`

The current editable house document lives in `house_state.state` as JSONB. Relational tables cover identity, sessions, permissions, audit history, AI runs, proposals, and future MCP/API access.

## Static Container

For a frontend-only smoke test of the new Vue app with Podman:

```powershell
.\scripts\serve.ps1
```

Then visit `http://localhost:8080`. API calls will fall back to seed data unless an API is available behind the Nginx `/api` proxy.
