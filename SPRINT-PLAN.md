# Sprint 9 — Security Hardening (Days 46–50)

**Goals**

* One-time boot tokens (MAC-bound, TTL), TLS, secrets minimalism.
* Age keys for bundle signing; token rotation for agents.

**Tasks**

1. Add token store to API (in-memory → Postgres with TTL).
2. Enforce HTTPS everywhere (Ingress TLS; internal service env toggles).
3. Implement agent token refresh endpoint.

**Acceptance**

* Boot tokens expire upon first use or TTL; re-use fails.
* All ingress are TLS; agents can rotate token without full reinstall.

**Codex Prompt:**

```
Harden auth:

1) services/api/api.go: add TokenStore backed by Postgres with schema tokens(id uuid pk, mac text, token text unique, expires_at timestamptz, used bool default false).
2) routes.go:
- Issue token in /v1/boot/ipxe path if none active; mark used on first render.
- POST /v1/agents/token/refresh {machine_id, old_token} -> returns new token with rotated expiry; invalidate old_token.

3) Update Helm ingress templates for TLS termination with a self-signed secret placeholder.

Provide full diff for updated files and token SQL migration 0002_tokens.sql.
```

# Sprint 10 — Polishing & Docs (Days 51–55)

**Goals**

* README completeness, `values-dev.yaml` wiring, example infra profiles, quickstart guide.
* Smoke tests.

**Tasks**

1. Flesh out `README.md` with quickstart, dev, lab notes.
2. Add example `infra/` profiles for two RHEL and one Windows machine.
3. Add a `scripts/smoke.sh` to deploy and hit health endpoints.
4. Ensure `helm lint` & `golangci-lint` clean.

**Acceptance**

* New dev can clone → `make tidy && helm upgrade --install` → working stack in <10 minutes (assuming S3/NATS/PG charts included).
* Smoke tests green.

**Codex Prompt:**

```
Finalize developer experience:

1) Update README.md with:
- Prereqs
- Quickstart for Docker Desktop Kubernetes
- How to set env (DB_DSN, NATS_URL, S3_*), how to render kickstart/unattend
- Air-gap bundle build/import
- PXE in lab vs dev caveats

2) Create build/scripts/smoke.sh bash script that:
- checks /healthz of every service via kubectl port-forward
- asserts non-200 fails the script.

3) Add helm values-dev.yaml defaults for DB_DSN, NATS_URL, S3 endpoint pointing to local charts; include how to install those charts.

Return full file contents for README.md and smoke.sh.
```

## Optional After Sprints

* **DHCP/TFTP** plugin for `bootd` (ProxyDHCP), only used in lab.
* **Secure Boot** flow (signed iPXE or shim).
* **UI** (Next.js/HTMX) after headless stabilizes.
* **Repo mirrors** (RHEL BaseOS/AppStream snapshot), Windows driver catalog builder.
* **TPM attestation** gate for agent registration.

---

### Daily Ritual (repeat each sprint)

* `make tidy && make lint && make test`
* `helm dependency build deploy/helm/umbrella && helm upgrade --install goose ...`
* `kubectl -n goose logs -f deploy/<svc>`
* `Update dashboards as you add new metrics.`