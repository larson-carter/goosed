# Sprint 0 — Bootstrap & Plumbing (Days 1–3)

**Goals**

* Compile & run skeleton services with health/metrics.
* Devcontainer, Makefile, go.work.
* Helm umbrella installs to Docker Desktop Kubernetes (DDK).
* OpenTelemetry wire-up (traces/logs/metrics).

**Tasks**

1. Fill `go.work` to include all submodules under `services/` and `pkg/`.
2. Implement `pkg/telemetry` with OTLP exporter (env: `OTEL_EXPORTER_OTLP_ENDPOINT`).
3. Create minimal `main.go` for each service with `/healthz`, `/readyz`, `/metrics`.
4. Provide `Makefile` targets (`tidy`, `lint`, `test`, `build`, `run-api`, `run-all`).
5. Write Helm charts (Deployment, Service, Ingress for `goosed-api`).
6. Add pre-push hook (`build/scripts/pre-push.sh`) to block dirty tree.

**Acceptance**

* `helm upgrade --install goose ...` runs pods for api/bootd/orchestrator/blueprints/inventory/artifacts-gw.
* `curl` each service `/healthz` → 200; Prom/OTel endpoints OK.

**Codex Prompt (paste):**

```
You are coding inside the goosed monorepo.

1) Create pkg/telemetry/telemetry.go with:
- Init(ctx, serviceName string) (shutdown func(context.Context) error, middleware func(http.Handler) http.Handler, logger *log.Logger, err error)
- Uses OTEL_EXPORTER_OTLP_ENDPOINT for OTLP HTTP export, W3C trace propagation, and structured JSON logs {ts,level,service,msg,trace_id}.

2) For each service entrypoint:
- services/api/cmd/api/main.go
- services/bootd/cmd/bootd/main.go
- services/orchestrator/cmd/orchestrator/main.go
- services/blueprints/cmd/blueprints/main.go
- services/inventory/cmd/inventory/main.go
- services/artifacts-gw/cmd/artifacts-gw/main.go
Add an HTTP server :8080, handlers /healthz(200), /readyz(200), /metrics (Prometheus), mount telemetry middleware.

3) Root Makefile:
targets: tidy (go mod tidy in all modules), lint (golangci-lint run or no-op), test (go test ./... recursively), build (docker build each service Dockerfile), run-api (go run services/api/cmd/api/main.go), run-all (just echo for now).

4) deploy/helm/goosed-api:
- Chart.yaml, values.yaml, templates/{deployment.yaml,svc.yaml,ingress.yaml}
Deployment env includes OTEL_EXPORTER_OTLP_ENDPOINT, pod has liveness/readiness for /healthz,/readyz, port 8080; Ingress host api.goose.local.

Return full file contents for the above (create/overwrite).
```

# Sprint 1 — Database, Event Bus, S3 (Days 4–7)

**Goals**

* Postgres DSN from env; run migrations.
* NATS JetStream wrapper (publish/subscribe durable).
* S3 client for Ceph/SeaweedFS endpoints.

**Tasks**

1. Implement `pkg/db/db.go`, migrations in `pkg/db/migrations/0001_init.sql`.
2. Implement `pkg/bus/bus.go` (NATS URL env: `NATS_URL`).
3. Implement `pkg/s3/s3.go` using AWS SDK v2 + custom endpoint (`S3_ENDPOINT`, `S3_REGION`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_TLS`).

**Acceptance**

* API connects to PG, runs migration.
* Publish/subscribe test subject `goosed.test` flows through NATS.
* S3 Put/PresignGet works against your endpoint.

**Codex Prompt:**

```
Implement storage plumbing:

A) pkg/db/db.go:
- Open(ctx, dsn) with pgxpool.
- Migrate(ctx, pool) using pressly/goose and embedded SQL migrations in pkg/db/migrations.
- Helper Exec/Get/Select with context and timeouts.

B) pkg/db/migrations/0001_init.sql:
CREATE TABLES:
blueprints(id uuid primary key, name text, os text, version text, data jsonb, created_at timestamptz default now(), updated_at timestamptz default now());
workflows(id uuid primary key, name text, steps jsonb, created_at timestamptz default now(), updated_at timestamptz default now());
machines(id uuid primary key, mac text unique, serial text, profile jsonb, created_at timestamptz default now(), updated_at timestamptz default now());
runs(id uuid primary key, machine_id uuid references machines(id), blueprint_id uuid references blueprints(id), status text, started_at timestamptz, finished_at timestamptz, logs text);
artifacts(id uuid primary key, kind text, sha256 text, url text, meta jsonb, created_at timestamptz default now());
facts(id uuid primary key, machine_id uuid references machines(id), snapshot jsonb, created_at timestamptz default now());
audit(id bigserial primary key, actor text, action text, obj text, details jsonb, at timestamptz default now());

C) pkg/bus/bus.go:
- type Bus with Publish(ctx, subj string, v any) error
- Subscribe(ctx, subj, durable string, fn func(ctx context.Context, data []byte) error) (io.Closer, error)
Use NATS JetStream with JSON encode/decode, durable consumers.

D) pkg/s3/s3.go:
- NewClientFromEnv() (*Client, error)
- PutObject(ctx, bucket, key string, r io.Reader, size int64, sha256 string) error
- PresignGet(ctx, bucket, key string, ttl time.Duration) (string, error)
Configure AWS SDK v2 with custom endpoint & static creds, toggle TLS by env.
Return full code.
```

# Sprint 2 — API Contracts & Render Endpoints (Days 8–12)

**Goals**

* Headless API for machines, runs, artifacts, and renderers.
* OpenAPI stub.

**Tasks**

1. Create `services/api/api.go` types & DB wires.
2. Implement `services/api/routes.go` using `chi`.
3. Add templates in `pkg/render/templates/`.
4. Wire S3 presign for artifact uploads.

**Acceptance**

* `POST /v1/machines` upserts & emits NATS `goosed.machines.enrolled`.
* `/v1/boot/ipxe?mac=` renders iPXE with one-time token (UUID).
* `/v1/render/kickstart` & `/v1/render/unattend` render from templates.
* `POST /v1/artifacts` returns presigned PUT URL (or presigned GET proxy if you prefer upload external).

**Codex Prompt:**

```
Build API features.

1) services/api/api.go:
Define models (json, db tags):
Machine {ID uuid, MAC string, Serial string, Profile map[string]any, CreatedAt, UpdatedAt}
Run {ID uuid, MachineID uuid, BlueprintID uuid, Status string, StartedAt, FinishedAt, Logs string}
Artifact {ID uuid, Kind string, SHA256 string, URL string, Meta map[string]any, CreatedAt}
Blueprint {ID uuid, Name, OS, Version, Data map[string]any}
Provide Store struct with DB, S3, Bus fields.

2) services/api/routes.go (chi router):
POST /v1/machines -> upsert by MAC, publish NATS "goosed.machines.enrolled" {machine_id,mac}
GET /v1/boot/ipxe?mac= -> lookup machine; render pkg/render/templates/ipxe.tmpl with {Token, MAC, APIBase}; short TTL token (in-memory map for now)
GET /v1/render/kickstart?machine_id= -> render kickstart.tmpl with profile vars
GET /v1/render/unattend?machine_id= -> render unattend.xml.tmpl with profile vars
POST /v1/artifacts -> body {kind, sha256, meta}; insert DB, return {upload_url} using s3.PresignGet or PUT variant
POST /v1/agents/facts -> {machine_id, snapshot}; insert facts, publish "goosed.agent.facts"
POST /v1/runs/start -> create running run
POST /v1/runs/finish -> set status and logs

Return full code for api.go and routes.go.
```

# Sprint 3 — Bootd & Artifacts-GW (Days 13–16)

**Goals**

* iPXE chain support (HTTP); branding files served.
* Presign GET proxy; HTTP Range passthrough.

**Tasks**

1. Implement `services/bootd/http.go` (`/menu.ipxe`, `/branding/*`).
2. Implement `services/artifacts-gw/presign.go` (`/v1/presign/get?key=...`).
3. Helm values to expose both services.

**Acceptance**

* `GET /menu.ipxe?mac=...` returns script that `chain` loads API `/v1/boot/ipxe`.
* Large downloads respect `Range:` header (verify by curl `--range`).

**Codex Prompt:**

```
Implement:

A) services/bootd/http.go:
- GET /menu.ipxe?mac= -> returns:
#!ipxe
set api http://api.goose.local
chain ${api}/v1/boot/ipxe?mac=${mac}
- Serve /branding/* from embedded FS under infra/branding (use fs.Sub).

B) services/artifacts-gw/presign.go:
- GET /v1/presign/get?key=K&ttl=300 -> s3.PresignGet(bucket=env S3_BUCKET, key=K), returns JSON {url}
- Document and pass Range headers through nginx ingress (add annotation snippet in code comment).

Return full code and minimal Dockerfiles.
```

# Sprint 4 — Blueprints, Inventory, Orchestrator (Days 17–22)

**Goals**

* Blueprints service reads `infra/` dir; publishes `goosed.blueprints.updated`.
* Inventory ingests facts & stores diffs.
* Orchestrator reacts to events: enrolled → start run → mark success on agent facts completion flag.

**Tasks**

1. `services/blueprints/gitpull.go` (local dir watcher) + `renderer.go`.
2. `services/inventory/ingest.go` NATS consumer for `goosed.agent.facts`.
3. `services/orchestrator/sm.go` listens to `goosed.machines.enrolled`, `goosed.agent.facts`, `goosed.runs.*`.

**Acceptance**

* Simulate machine enroll → orchestrator creates a run and waits.
* POST facts with `{postinstall_done:true}` → run ends `success`.

**Codex Prompt:**

```
Code three services:

A) Blueprints
- gitpull.go: every 30s read infra/blueprints and infra/workflows from repo local path (env INFRA_PATH default "./infra"); cache in memory; publish "goosed.blueprints.updated" with version nonce.
- renderer.go: function RenderKickstart(profile map[string]any) string and RenderUnattend(profile map[string]any) string using pkg/render templates.

B) Inventory
- Subscribe to "goosed.agent.facts"; insert into facts table; compute diff from last snapshot (only top-level keys), store summary into audit table (actor="agent", action="facts_updated").

C) Orchestrator
- On "goosed.machines.enrolled": create run(status=running, started_at=now).
- On "goosed.agent.facts" with payload {machine_id, snapshot.postinstall_done=true}: set run finished(success) for latest running run of that machine.

Return complete code for these files with error handling and logging.
```
# Sprint 5 — RHEL Agent MVP (Days 23–27)

**Goals**

* RHEL `%post` installer + systemd agent that posts facts & completion.
* Sample Kickstart uses agent install snippet.

**Tasks**

1. `services/agents/rhel/postinstall.sh` to drop config and install unit.
2. `services/agents/rhel/service.go` posts basics every 30s; on first run, sets `postinstall_done=true`.
3. Add `pkg/render/templates/kickstart.tmpl` snippet to curl/install agent.

**Acceptance**

* In a VM, install via Kickstart URL → agent registers; facts visible; run finishes.

**Codex Prompt:**

```
Implement RHEL agent:

1) services/agents/rhel/postinstall.sh:
- Read env API_URL and TOKEN from kernel args or files.
- Create /etc/goosed/agent.conf JSON with {api, token, machine_id}
- Install systemd unit /etc/systemd/system/goosed-agent.service that runs /usr/local/bin/goosed-agent-rhel
- Enable & start.

2) services/agents/rhel/service.go:
- Load config; on start, POST /v1/agents/facts snapshot {kernel, selinux, packages: ["placeholder"], postinstall_done:true if first boot}
- Loop every 30s to send minimal heartbeat facts.

3) pkg/render/templates/kickstart.tmpl: add in %post:
curl -fsSL {{ .AgentInstallURL }} | bash -s -- --api {{ .APIBase }} --token {{ .Token }} --machine {{ .MachineID }}

Return file contents.
```

# Sprint 6 — Windows Agent MVP (Days 28–33)

**Goals**

* WinPE `provision.ps1` skeleton calls API and deploys agent service.
* Windows agent posts facts once.

**Tasks**

1. `services/agents/windows/provision.ps1` does DISM/WMI facts sample + registers agent.
2. `services/agents/windows/service.go` is a Windows service (golang.org/x/sys/windows/svc) posting snapshot.
3. Add `pkg/render/templates/unattend.xml.tmpl` to place agent config at first boot.

**Acceptance**

* In a Win11 VM, provision → service posts facts to API and `postinstall_done:true`.

**Codex Prompt:**

```
Create Windows agent:

1) services/agents/windows/provision.ps1:
- Param($Api, $Token, $MachineId)
- Collect WMI: OS caption, version, BIOS serial
- Invoke-RestMethod POST $Api/v1/agents/facts with JSON {machine_id, snapshot:{os, version, serial, postinstall_done:true}}
- Write C:\ProgramData\Goosed\agent.conf with api/token/machine_id
- Register service 'GoosedAgent' to run agent executable.

2) services/agents/windows/service.go:
- Implement basic Windows service that reads config and posts a heartbeat fact.

3) pkg/render/templates/unattend.xml.tmpl:
- Insert FirstLogonCommands to run powershell provisioning with API and token arguments.

Return code and templates.
```

# Sprint 7 — Artifacts, Bundler & Air-Gap (Days 34–40)

**Goals**

* `goosectl` to build/import bundles (tar.zst) with manifest signing (age/ed25519).
* `artifacts-gw` upload/download UX polished.

**Tasks**

1. `services/bundler/bundler.go` & `signer.go`: build/import; sign/verify manifest.
2. Extend API `/v1/artifacts` to accept register-only vs presign mode.
3. Document in README the offline flow.

**Acceptance**

* `goosectl bundles build --artifacts-dir ./iso --output bundle-*.tar.zst` produces signed bundle.
* `goosectl bundles import --file ...` registers and uploads to S3; API list shows artifacts.

**Codex Prompt:**

```
Add a bundler CLI:

A) services/bundler/cmd/goosectl/main.go:
- cobra-based CLI with cmd: bundles build, bundles import
- flags: --artifacts-dir, --images-file, --output for build; --file, --api for import

B) services/bundler/bundler.go:
- Build: walk artifacts-dir, compute sha256 for each file, create manifest.yaml with entries {path, sha256, size, kind inferred by extension}, tar.zst writer producing bundle file; then sign manifest via signer.

C) services/bundler/signer.go:
- Use age with env AGE_SECRET_KEY to sign manifest bytes; output signature in manifest as base64. Verify on import.

D) Import path: read bundle, verify signature, upload objects to S3 using s3.PutObject, POST /v1/artifacts per file to register.

Return all code files complete.
```

# Sprint 8 — Observability & Dashboards (Days 41–45)

**Goals**

* OTel Collector config + Prom/Loki/Tempo charts in `goosed-observability`.
* Grafana dashboards for API latency, workflow durations, agent health.

**Tasks**

1. Fill `ops/otel/collector.yaml`, Prom scrape configs, Loki & Tempo configs.
2. Create dashboards JSONs in `ops/grafana/dashboards/`.
3. Helm chart `goosed-observability` to deploy stack with datasources from `ops/grafana/datasources.yaml`.

**Acceptance**

* Grafana shows: API p50/p95/p99, error rates; Orchestrator step histograms; Agent last-seen panel; S3 throughput.

**Codex Prompt:**

```
Produce observability assets:

1) ops/otel/collector.yaml: receivers (otlp http:4318), processors (batch), exporters:
- prometheus (0.0.0.0:9464) OR prometheusremotewrite disabled, 
- tempo (otlp to tempo),
- logging (info)
Include service pipelines for traces, metrics, logs.

2) deploy/helm/goosed-observability:
- Chart that deploys otel-collector (Deployment + Service), Prometheus, Loki, Tempo, Grafana with datasources from ops/grafana/datasources.yaml.
- Grafana loads dashboards from configmap mounted at /var/lib/grafana/dashboards.

3) ops/grafana/dashboards/api.json: panels for http_server_duration_seconds histogram (p95), request count, error rate; label by service="goosed-api".
Create similar dashboards for orchestrator, bootd, agents.

Return all YAML/JSON contents.
```

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
