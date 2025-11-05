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