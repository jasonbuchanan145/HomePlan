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

The app starts with no house in local dev mode. Seed the deterministic user 1 demo house with:

```powershell
.\scripts\seed-dev-house.ps1
```

Reset and reseed it with:

```powershell
.\scripts\seed-dev-house.ps1 -Reset
```

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

The frontend tries `GET /api/house/current` first. If no house is available, it starts from an empty state and lets the homeowner create a blank or guided local draft before saving.

Native Windows `npm` is not the primary verification path on the current development machine. Prefer the Minikube image build for frontend verification:

```powershell
minikube image build --build-opt no-cache=true -t localhost/homeplan-web:verify-room-drafts -f web/Containerfile .
```

## Playwright

The first Playwright suite lives in `web/e2e` and targets the Minikube-served app at `http://localhost:8080`.

```powershell
cd web
npm run test:e2e
```

Set `PLAYWRIGHT_BASE_URL` to point at another running deployment. The tests assume the local dev endpoints are available so each run can reset the deterministic dev house.

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

Then visit `http://localhost:8080`. Without an API behind the Nginx `/api` proxy, the app stays in its empty/local draft flow and saves drafts in the browser until the API is available.
