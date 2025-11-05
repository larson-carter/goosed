# goose’d 🪿

**Golang Operating System Environment Deployer** — a headless-first, Git-driven PXE platform for **RHEL** and **Windows** provisioning designed for **air-gapped labs**.

## TL;DR

* **One platform** for RHEL + Windows imaging/install flows
* **Headless-first**: CLI + API now, UI later
* **Git as source of truth**: blueprints/workflows in Git; facts & run history tracked
* **Air-gap ready**: import/export signed bundles to offline S3 (Ceph RGW / SeaweedFS)
* **Custom boot UX**: iPXE/GRUB/pxelinux theming + one-time boot tokens
* **Observability**: OpenTelemetry → Prometheus/Loki/Tempo → Grafana

## Table of Contents

1. [Architecture](#architecture)
2. [Microservices](#microservices)
3. [Tech Stack](#tech-stack)
4. [Repository Layout](#repository-layout)
5. [Getting Started (Dev on Docker Desktop K8s)](#getting-started-dev-on-docker-desktop-k8s)
6. [Configuration & Environment](#configuration--environment)
7. [Deploying the Stack](#deploying-the-stack)
8. [PXE Boot: Dev vs Lab](#pxe-boot-dev-vs-lab)
9. [RHEL & Windows Provisioning Flows](#rhel--windows-provisioning-flows)
10. [Air-Gap Bundles (`goosectl`)](#air-gap-bundles-goosectl)
11. [Observability](#observability)
12. [Security](#security)
13. [API Overview](#api-overview)
14. [GitOps (`infra/`) Layout](#gitops-infra-layout)
15. [Development Workflow](#development-workflow)
16. [Makefile Targets](#makefile-targets)
17. [Troubleshooting](#troubleshooting)
18. [Roadmap](#roadmap)

## Architecture

goose’d is a set of Go microservices (running as containers) plus a small set of shared libraries:

* **Data plane**: PXE/iPXE/HTTPBoot, artifacts (kernels, initrds, WIMs, ISOs), Kickstart/Unattend rendering.
* **Control plane**: API, workflow orchestrator, blueprint renderer, inventory/facts, artifacts gateway.
* **State**: Postgres (core DB), NATS JetStream (events), S3-compatible storage (artifacts).
* **Agent**: RHEL systemd service & Windows service to execute post-install ops and report facts.
* **Observability**: OTel → Prometheus (metrics), Loki (logs), Tempo (traces), Grafana (dashboards).

Everything is **headless** (API + CLI). Add the UI later without blocking provisioning.

## Microservices

| Service                  | What it does                                                                                                       | Depends on                  |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------ | --------------------------- |
| **api**                  | REST for machines, runs, artifacts, render endpoints (iPXE/Kickstart/Unattend), audit; issues one-time boot tokens | Postgres, NATS, S3          |
| **bootd**                | PXE edge for **iPXE/HTTPBoot**; serves branding; chains to API render endpoints                                    | API, S3, NATS               |
| **orchestrator**         | Workflow state machine (Boot → Provision → Post → Verify → Report)                                                 | API, NATS, Git (read infra) |
| **blueprints**           | Pulls/reads `infra/` blueprints & workflows, renders templates, emits update events                                | Git, API, NATS              |
| **inventory**            | Consumes agent facts, stores snapshots, computes diffs                                                             | API, NATS                   |
| **artifacts-gw**         | S3 presign proxy, optional range-request passthrough for large objects                                             | S3, API                     |
| **bundler** (`goosectl`) | Build/import **air-gap bundles** (images, ISOs/WIMs, drivers); sign & verify                                       | S3, API                     |
| **agent-rhel**           | Systemd service for post-install ops & facts                                                                       | API, S3                     |
| **agent-windows**        | Windows service (PowerShell bootstrap) for post-install & facts                                                    | API, S3                     |

**Event bus**: subjects like `goosed.machines.enrolled`, `goosed.runs.started`, `goosed.agent.facts`, `goosed.blueprints.updated`.

## Tech Stack

* **Language**: Go 1.22+
* **DB**: Postgres 16 (JSONB)
* **Events**: NATS JetStream
* **Artifacts**: **SeaweedFS S3** (dev) / **Ceph RGW** (lab)
* **Tracing/Logs/Metrics**: OpenTelemetry → Tempo/Loki/Prometheus → Grafana
* **Kubernetes**: Docker Desktop Kubernetes (dev) & air-gapped K8s/VMs (lab)
* **Templates**: Go `text/template` for Kickstart, Unattend, iPXE

## Repository Layout

```
goosed/
├─ README.md
├─ .devcontainer/                 # VS Code devcontainer
├─ build/                         # base Dockerfile, scripts & hooks
├─ deploy/
│  ├─ helm/
│  │  ├─ umbrella/               # umbrella chart
│  │  ├─ goosed-api/ …           # per-service charts
│  │  └─ goosed-observability/
│  └─ k8s/                       # cluster-level bits (ns, ingressclass)
├─ ops/                           # Observability configs (Grafana/OTel/Prom/Loki/Tempo)
├─ pkg/                           # shared libs: bus (NATS), s3, db, telemetry, render, auth
├─ services/
│  ├─ api/                        # REST + renders
│  ├─ bootd/                      # iPXE/HTTPBoot edge
│  ├─ orchestrator/               # workflows
│  ├─ blueprints/                 # git pull + renderer
│  ├─ inventory/                  # facts ingestion
│  ├─ artifacts-gw/               # presign gateway
│  ├─ bundler/                    # goosectl (CLI) + bundle logic
│  └─ agents/
│     ├─ rhel/                    # systemd agent + %post
│     └─ windows/                 # Windows service + WinPE bootstrap
└─ infra/                         # GitOps desired state (blueprints/workflows/profiles/branding/policies)
```

## Getting Started (Dev on Docker Desktop K8s)

**Prereqs**

* Docker Desktop with Kubernetes enabled
* Helm 3, kubectl, Go 1.22+
* (Optional) VS Code Dev Containers

**1) Clone & open**

```bash
git clone <your_repo> goosed
cd goosed
```

**2) (Optional) Devcontainer**

* Open in VS Code → “Reopen in Container”.

**3) Build base + tidy**

```bash
make tidy
```

**4) Provide local infra**

* Install Postgres, NATS, and S3 (SeaweedFS S3) in your cluster or point to existing endpoints.
  *You can also add them as subcharts in the umbrella values for quick dev.*

**5) Deploy**

```bash
helm upgrade --install goose deploy/helm/umbrella \
  -n goose --create-namespace \
  -f deploy/helm/umbrella/values-dev.yaml

kubectl -n goose get pods
```

**6) Smoke check**

```bash
kubectl -n goose port-forward svc/goosed-api 8080:8080 & sleep 1
curl -f localhost:8080/healthz
```

## Configuration & Environment

Common envs (surfaced via Helm values):

* **Postgres**: `DB_DSN=postgres://user:pass@postgres.pg:5432/goose?sslmode=disable`
* **NATS**: `NATS_URL=nats://nats.nats:4222`
* **S3**:

    * `S3_ENDPOINT=http://s3.s3:8333`
    * `S3_REGION=lab`
    * `S3_BUCKET=goose-artifacts`
    * `S3_ACCESS_KEY=...` / `S3_SECRET_KEY=...`
    * `S3_TLS=false`
* **OTel**: `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector.obsv:4318`

Ingress hosts (dev):

* `api.goose.local` (map to 127.0.0.1 via `/etc/hosts` if needed)

## Deploying the Stack

**Umbrella chart**

```bash
helm dependency build deploy/helm/umbrella
helm upgrade --install goose deploy/helm/umbrella -n goose \
  -f deploy/helm/umbrella/values-dev.yaml
```

**Per-service charts** live under `deploy/helm/<service>/` and are referenced by the umbrella.

## PXE Boot: Dev vs Lab

* **Dev (Docker Desktop K8s)**: Broadcast DHCP/TFTP is hard; prefer **HTTPBoot/iPXE**. Run DHCP externally or use a small VM.
* **Lab (air-gapped)**: Run **bootd** on a **bridged** host/VM on the PXE VLAN with **ProxyDHCP + TFTP** (optional) and HTTP for large artifacts. Point bootd to the API and S3 endpoints.

> UEFI Secure Boot: sign iPXE or use a trusted shim if you need Secure Boot enabled.

## RHEL & Windows Provisioning Flows

### RHEL (Kickstart)

1. PXE → iPXE → `GET /v1/boot/ipxe?mac=...` (API) → dynamic ks URL
2. Kickstart renders with repo mirrors, partitioning, users, `%post` installs **agent-rhel**
3. First boot: agent runs ops (packages, hardening), posts **facts**, orchestrator marks **run** done

**Kickstart template**: `pkg/render/templates/kickstart.tmpl`

### Windows (WinPE/Unattend)

1. iPXE + **wimboot** loads WinPE (HTTP)
2. `provision.ps1` fetches Unattend, DISM /Apply-Image, injects drivers, configures **agent-windows**
3. Agent posts **facts**; orchestrator completes run

**Unattend template**: `pkg/render/templates/unattend.xml.tmpl`
**WinPE script**: `services/agents/windows/provision.ps1`

## Air-Gap Bundles (`goosectl`)

Bundle content: images, ISOs/WIMs, drivers, metadata manifest. Signed with **age/Ed25519**.

**Build**

```bash
go run ./services/bundler/cmd/goosectl \
  bundles build \
  --artifacts-dir ./artifacts \
  --images-file ./images.txt \
  --output ./bundle-$(date +%Y%m%d).tar.zst
```

**Import**

```bash
go run ./services/bundler/cmd/goosectl \
  bundles import \
  --file ./bundle-20251104.tar.zst \
  --api https://api.goose.local
```

The import uploads to S3 and registers artifacts via the API.

---

## Observability

* **OTel Collector** (`ops/otel/collector.yaml`) receives service traces/metrics/logs.
* **Prometheus** scrapes `/metrics` on each service.
* **Loki** ingests JSON logs.
* **Tempo** stores traces.
* **Grafana** (dashboards in `ops/grafana/dashboards/`) visualizes:

    * API latency (p50/p95/p99), error rate
    * Orchestrator step timings
    * Agent last-seen, success ratio
    * Artifact throughput (S3)

Helm chart: `deploy/helm/goosed-observability`.

## Security

* **TLS** at ingress; internal services honor TLS toggles.
* **One-time boot tokens** bound to MAC/serial with short TTL, marked used on first render.
* **Bundle signing** via age/Ed25519; verify on import.
* **Least secrets** on disk; agent tokens rotate.
* Optional: TPM attestation gate before agent registration (post-MVP).

## API Overview

Key endpoints (see `services/api/openapi.yaml`):

* `POST /v1/machines` — enroll/upsert machine `{mac, serial, profile}`
* `GET /v1/boot/ipxe?mac=...` — render iPXE script with one-time token
* `GET /v1/render/kickstart?machine_id=...` — render Kickstart
* `GET /v1/render/unattend?machine_id=...` — render Unattend
* `POST /v1/artifacts` — register artifact & return presigned URL
* `POST /v1/agents/facts` — store facts snapshot & emit event
* `POST /v1/runs/start|finish` — run state transitions

## GitOps (`infra/`) Layout

```
infra/
  blueprints/
    rhel/9/base/blueprint.yaml
    windows/11/base/blueprint.yaml
  workflows/
    rhel-default.yaml
    windows-default.yaml
  machine-profiles/
    lab-a/rack-01/01-mac-001122aabbcc.yaml
  branding/branding.yaml
  policies/cis/{rhel9.yaml, win11.yaml}
```

* Overlays: org → site → rack → node.
* `blueprints` + `workflows` drive renders; `machine-profiles` bind a machine to a blueprint and variables.
* Agent **facts** and run logs are stored in DB (optionally committed back to Git later).

## Development Workflow

* Use the **devcontainer** to get Go, kubectl, Helm, golangci-lint, etc.
* Implement features in `pkg/` libs first, then wire in services.
* Every service exposes `/healthz`, `/readyz`, and `/metrics`.
* Keep OpenTelemetry context propagation when calling each other and NATS.

## Makefile Targets

```make
tidy        # go mod tidy across modules
lint        # golangci-lint run (if installed)
test        # go test ./... across repo
build       # docker build each service
run-api     # go run services/api/cmd/api/main.go
deploy      # helm upgrade --install umbrella (values-dev)
undeploy    # helm uninstall
```

## Troubleshooting

**Ingress 404**

* Check IngressClass and hostnames in `values-dev.yaml` (e.g., `api.goose.local` in `/etc/hosts`).

**Can’t PXE in Docker Desktop**

* Use iPXE/HTTPBoot and external DHCP. In lab, run **bootd** on bridged host/VM.

**Large WIM/ISO slow or partial**

* Ensure **Range** headers pass through ingress to **artifacts-gw** and S3.

**No metrics in Grafana**

* Verify `OTEL_EXPORTER_OTLP_ENDPOINT` env; Prometheus scrape annotations on services.

**NATS consumers not receiving**

* Check JetStream stream/consumer creation and `durable` names; confirm `NATS_URL`.

## Roadmap

* DHCP/TFTP (ProxyDHCP) module for **bootd** (lab only)
* Secure Boot flow (signed iPXE/shim)
* UI (Next.js/HTMX)
* Repo mirrors (RHEL BaseOS/AppStream) & Windows driver catalog
* TPM attestation before agent registration
* Domain join (offline secrets blob) & CIS baselines library
