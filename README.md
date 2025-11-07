# goose’d 🪿

**Golang Operating System Environment Deployer** — a headless-first, Git-driven PXE platform for **RHEL** and **Windows** provisioning designed for **air-gapped labs**.

## TL;DR

* **One platform** for RHEL/Rocky + Windows imaging/install flows
* **Headless-first**: CLI + API now, UI later
* **Git as source of truth**: blueprints/workflows in Git; facts & run history tracked
* **Air-gap ready**: import/export signed bundles to offline S3 (SeaweedFS everywhere, mirror as needed)
* **Custom boot UX**: iPXE/GRUB/pxelinux theming + one-time boot tokens
* **Observability**: OpenTelemetry → Prometheus/Loki/Tempo → Grafana

## Table of Contents

1. [How to Use This README](#how-to-use-this-readme)
2. [Architecture](#architecture)
3. [Microservices](#microservices)
4. [Tech Stack](#tech-stack)
5. [Repository Layout](#repository-layout)
6. [Documentation Index](#documentation-index)
7. [Observability](#observability)
8. [Security](#security)
9. [API Overview](#api-overview)
10. [GitOps (`infra/`) Layout](#gitops-infra-layout)
11. [Development Workflow](#development-workflow)
12. [Makefile Targets](#makefile-targets)
13. [Troubleshooting](#troubleshooting)
14. [Roadmap](#roadmap)

## How to Use This README

1. **Start with the TL;DR** above for the high-level pitch, then jump back here when you need specifics.
2. **Use the Table of Contents** to hop directly to the area you care about—each major activity (dev setup, deployments, PXE flows) has its own section.
3. **Follow callouts** such as _Prereqs_, numbered walkthroughs, and code fences to complete tasks end-to-end without hunting elsewhere.
4. **Cross-reference sprint notes** in `SPRINT-PLAN.md` when you need historical context or validation checklists for recently completed work.
5. **Search-friendly tips**: `git grep`/`rg` the headings if you are in an editor, or view this README in VS Code’s outline for quick navigation.

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
| **pxe-stack**            | DHCP/TFTP helpers (deployed by default) to hand out iPXE binaries and bridge into the API
                | Host network, branding data  |
| **blueprints**           | Pulls/reads `infra/` blueprints & workflows, renders templates, emits update events                                | Git, API, NATS              |
| **inventory**            | Consumes agent facts, stores snapshots, computes diffs                                                             | API, NATS                   |
| **artifacts-gw**         | S3 presign proxy, optional range-request passthrough for large objects                                             | S3, API                     |
| **bundler** (`goosectl`) | Build/import **air-gap bundles** (images, ISOs/WIMs, drivers); sign & verify                                       | S3, API                     |
| **agent-rhel**           | Systemd service for post-install ops & facts                                                                       | API, S3                     |
| **agent-windows**        | Windows service (PowerShell bootstrap) for post-install & facts                                                    | API, S3                     |

**Event bus**: subjects like `goosed.machines.enrolled`, `goosed.runs.started`, `goosed.agent.facts`, `goosed.blueprints.updated`.

## Tech Stack

* **Language**: Go 1.25+
* **DB**: Postgres 17 (JSONB) via `pgxpool` + `gorm` migrations orchestrated by `pressly/goose`
* **Events**: NATS JetStream
* **Artifacts**: **SeaweedFS S3** (dev & prod)
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

## Documentation Index

* [Getting Started (Prereqs + Quickstart)](docs/getting-started.md)
* [Deploying goose'd in Kubernetes](docs/deploying.md)
* [Observability with Grafana](docs/observability.md)
* [PXE Boot Strategies: Development vs Lab](docs/pxe-environments.md)
* [PXE Booting VMware Fusion Guests](docs/vmware-fusion.md)
* [RHEL/Rocky and Windows Provisioning Flows](docs/provisioning-flows.md)
* [Building and Importing Air-Gap Bundles](docs/air-gap-bundles.md)
* [Uploading Rocky Linux ISOs to SeaweedFS](docs/seaweedfs-iso-upload.md)

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
    rocky/9/base/blueprint.yaml
    windows/11/base/blueprint.yaml
  workflows/
    rhel-default.yaml
    rocky-default.yaml
    windows-default.yaml
  machine-profiles/
    lab-a/rack-01/01-mac-001122aabbcc.yaml
    lab-a/rack-01/03-mac-001122ccddee.yaml
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
