# goose'd UI Service

The goose'd UI ships as two containers that share a Helm chart:

* **ui-api** – a Go service that owns user/session data, issues JWTs, and exposes REST hooks for the web frontend. It stores
  data in the `ui_auth` schema of the main Postgres instance and is configured entirely through environment variables.
* **ui-web** – a static Vite/React build published via Nginx. It renders authentication screens, the dashboard shell, and
  placeholder resource views while the API endpoints are implemented.

Both components live under `services/ui/` in this repository. The top-level chart `services/ui/chart/goosed-ui` deploys them in a
single pod by default so you can port-forward one service for both the API and the web frontend.【F:services/ui/chart/goosed-ui/values.yaml†L1-L35】【F:services/ui/chart/goosed-ui/templates/deploy.yaml†L1-L48】

## Runtime configuration

The UI API relies on conventional environment variables for its dependencies and cookies. The table below maps the most
important variables and their defaults. Override anything marked _required_ before booting the service.【F:services/ui/api/internal/config/config.go†L10-L34】

| Variable | Purpose | Notes |
| --- | --- | --- |
| `DB_DSN` | Postgres connection string | Required; UI API stores data under the `ui_auth` schema and will create it on start-up.【F:services/ui/api/internal/db/db.go†L12-L35】 |
| `JWT_SIGNING_KEY` / `JWT_REFRESH_KEY` | HMAC secrets used to mint access/refresh tokens | Required; generate long, random values for both keys. |
| `API_BASE_URL` | Base URL for the core goose'd API | Used when the UI needs to call back into the provisioning API. |
| `CORS_ALLOWED_ORIGINS` | Allowed browser origins | Defaults to `http://localhost:5173` for local development. |
| `COOKIE_DOMAIN` / `COOKIE_SECURE` | Session cookie settings | Scope cookies to your domain; set `COOKIE_SECURE=true` when TLS terminates before the pod. |
| `ACCESS_TOKEN_TTL` / `REFRESH_TOKEN_TTL` | Lifetimes for issued JWTs | Defaults to 15 minutes and 14 days respectively. |

All remaining fields (SMTP, OIDC, OTLP) are optional and can be left unset until those integrations are wired up.

## Building images for local registries

The repo includes a helper target that builds both containers with a single command. Provide the image tags you want to push to
kind, Docker Desktop, or your registry of choice.【F:Makefile†L1-L29】【F:Makefile†L37-L45】

```bash
make ui-build UI_API_IMAGE=goosed/ui-api:dev UI_WEB_IMAGE=goosed/ui-web:dev
```

## Deploying on Kubernetes

The Helm chart defaults to a `ClusterIP` service exposing port `80` for the web UI and `8080` for the API. Keep the single-pod
strategy for development; switch `podStrategy.singlePod=false` if you want to scale the API and web tiers separately.【F:services/ui/chart/goosed-ui/values.yaml†L1-L35】【F:services/ui/chart/goosed-ui/templates/svc.yaml†L1-L34】

1. Generate fresh signing keys:

   ```bash
   UI_JWT_SIGNING_KEY=$(openssl rand -base64 32)
   UI_JWT_REFRESH_KEY=$(openssl rand -base64 32)
   ```

2. Install or upgrade the chart using the dev Postgres DSN from the umbrella stack and your API ingress host:

   ```bash
   helm upgrade --install goosed-ui services/ui/chart/goosed-ui \
     --namespace goose \
     --set config.env.API_BASE_URL="https://api.goosed.local" \
     --set config.env.CORS_ALLOWED_ORIGINS="http://localhost:5173" \
     --set config.secret.DB_DSN="postgres://goosed:goosed@goose-postgres-postgresql.goose.svc.cluster.local:5432/goosed?sslmode=disable" \
     --set config.secret.JWT_SIGNING_KEY="${UI_JWT_SIGNING_KEY}" \
     --set config.secret.JWT_REFRESH_KEY="${UI_JWT_REFRESH_KEY}"
   ```

3. Port-forward the service to reach the web frontend:

   ```bash
   kubectl -n goose port-forward svc/goosed-ui 5173:80
   ```

The API is exposed on the same `svc/goosed-ui` resource (`8080/TCP`) when you need to inspect health endpoints or wire up
integration tests.【F:services/ui/chart/goosed-ui/templates/svc.yaml†L1-L34】

## Accessing the preview UI

Browse to `http://localhost:5173/auth/login` once the port-forward is active. The login form, forgot password, and invite
screens are implemented in the web bundle but do not yet make API calls—the backend currently exposes health, readiness, metrics
and a placeholder server-sent events endpoint only.【F:services/ui/web/src/pages/auth/login.tsx†L1-L44】【F:services/ui/api/internal/handlers/router.go†L1-L48】

You can still navigate around the dashboard shell to preview the UX. Authentication flows will light up as the UI API gains the
remaining endpoints for invites, password resets, and sessions.

## Related references

* Helm chart: `services/ui/chart/goosed-ui`
* UI API code: `services/ui/api`
* Web frontend: `services/ui/web`
* Make targets: `make ui-build`
