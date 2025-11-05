# ~~Sprint 2 — API Contracts & Render Endpoints (Days 8–12)~~

**Goals**

* ~~Headless API for machines, runs, artifacts, and renderers.~~
* ~~OpenAPI stub.~~

**Tasks**

1. ~~Create `services/api/api.go` types & DB wires.~~
2. ~~Implement `services/api/routes.go` using `chi`.~~
3. ~~Add templates in `pkg/render/templates/`.~~
4. ~~Wire S3 presign for artifact uploads.~~

**Acceptance**

* ~~`POST /v1/machines` upserts & emits NATS `goosed.machines.enrolled`.~~
* ~~`/v1/boot/ipxe?mac=` renders iPXE with one-time token (UUID).~~
* ~~`/v1/render/kickstart` & `/v1/render/unattend` render from templates.~~
* ~~`POST /v1/artifacts` returns presigned PUT URL (or presigned GET proxy if you prefer upload external).~~

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
