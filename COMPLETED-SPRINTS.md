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
