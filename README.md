# goose’d 🪿

**Golang Operating System Environment Deployer** — a headless-first, Git-driven PXE platform for **RHEL** and **Windows** provisioning designed for **air-gapped labs**.

## TL;DR

* **One platform** for RHEL + Windows imaging/install flows
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
6. [Prerequisites](#prerequisites)
7. [Quickstart (Docker Desktop Kubernetes)](#quickstart-docker-desktop-kubernetes)
8. [Configuration & Environment](#configuration--environment)
9. [Rendering Kickstart & Unattend](#rendering-kickstart--unattend)
10. [Deploying the Stack](#deploying-the-stack)
11. [PXE Boot: Dev vs Lab](#pxe-boot-dev-vs-lab)
12. [RHEL & Windows Provisioning Flows](#rhel--windows-provisioning-flows)
13. [Air-Gap Bundles (`goosectl`)](#air-gap-bundles-goosectl)
14. [Observability](#observability)
15. [Security](#security)
16. [API Overview](#api-overview)
17. [GitOps (`infra/`) Layout](#gitops-infra-layout)
18. [Development Workflow](#development-workflow)
19. [Makefile Targets](#makefile-targets)
20. [Troubleshooting](#troubleshooting)
21. [Roadmap](#roadmap)

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
| **orchestrator**         | Workflow state machine (Boot → Provision → Post → Verify → Report)                                                 | API, NATS, Git (read infra) |
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

## Prerequisites

Before you try to run goose’d locally make sure you have the following pieces in place:

* **Docker Desktop 4.x with Kubernetes enabled** (or another single-node cluster such as kind/minikube).
* **kubectl 1.29+** with a context that points at your local cluster.
* **Helm 3.13+** for chart installation.
* **Go 1.25+**, **make**, **git**, **bash**, **curl**, **jq**, and the Docker Compose plugin (used by `setup-env.sh`).
* **golangci-lint v1.61.0+** so `make lint` matches CI behaviour.
* Optional but convenient: **VS Code + Dev Containers** (the repo ships `.devcontainer/`).
* Outbound internet access the first time you pull container images and Helm charts.

## Quickstart (Docker Desktop Kubernetes)

1. **Clone the repository**

   ```bash
   git clone <your_repo> goosed
   cd goosed
   ```

2. **Bootstrap local tooling** – this pulls development containers, generates `.env.development`, and verifies Postgres/NATS/SeaweedFS reachability on the host.

   ```bash
   ./setup-env.sh
   source .env.development
   ```

3. **Prepare Go modules**

   ```bash
   make tidy
   ```

4. **Install backing services into Kubernetes** (run once per cluster). The commands below create the `goose` namespace and install Postgres, NATS JetStream, and SeaweedFS with credentials that match `values-dev.yaml`.

   ```bash
   kubectl create namespace goose --dry-run=client -o yaml | kubectl apply -f -

   helm repo add bitnami https://charts.bitnami.com/bitnami
   helm repo add nats https://nats-io.github.io/k8s/helm/charts/
   helm repo add seaweedfs https://seaweedfs.github.io/seaweedfs/helm
   helm repo update

   helm upgrade --install goose-postgres oci://registry-1.docker.io/bitnamicharts/postgresql \
     --namespace goose \
     --set auth.username=goosed \
     --set auth.password=goosed \
     --set auth.database=goosed \
     --set primary.persistence.enabled=false

   helm upgrade --install goose-nats nats/nats \
     --namespace goose \
     --set replicaCount=1 \
     --set nats.jetstream.enabled=true \
     --set nats.auth.enabled=false

   helm upgrade --install goose-seaweedfs seaweedfs/seaweedfs \
     --namespace goose \
     --set master.replicaCount=1 \
     --set filer.replicaCount=1 \
     --set volume.replicaCount=1 \
     --set s3.enabled=true \
     --set s3.port=8333
   ```

5. **Build chart dependencies and deploy goose’d**

   ```bash
   helm dependency build deploy/helm/umbrella
   helm upgrade --install goose deploy/helm/umbrella \
     --namespace goose \
     -f deploy/helm/umbrella/values-dev.yaml
   ```

6. **Verify pods** – everything should settle into `Running`/`Completed` within a couple of minutes.

   ```bash
   kubectl -n goose get pods
   ```

7. **Run the smoke test** – this port-forwards each service and fails fast if any `/healthz` endpoint returns a non-200.

   ```bash
   ./build/scripts/smoke.sh
   ```

Add `api.goose.local`, `boot.goose.local`, and `artifacts.goose.local` to `/etc/hosts` (pointing to `127.0.0.1`) if you want to exercise the ingress paths in a browser.

The whole flow from clone → smoke test completes in well under ten minutes on a typical laptop once the initial container images are pulled.

## Configuration & Environment

`setup-env.sh` writes `.env.development` with sensible defaults for local binaries to talk to the backing services that run on your workstation:

| Variable | Purpose |
| --- | --- |
| `DB_DSN` | Postgres connection string used by CLI tools or local services (`postgres://goosed:goosed@localhost:5432/goosed?sslmode=disable`). |
| `NATS_URL` | JetStream endpoint (`nats://localhost:4222`). |
| `S3_ENDPOINT` | SeaweedFS S3 gateway (`http://localhost:8333`). |
| `S3_REGION` | Region hint for the S3 client (`us-east-1`). |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | Development credentials (`goosed` / `goosedsecret`). |
| `S3_DISABLE_TLS` | Set to `true` for plain HTTP during dev. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Local OTel collector endpoint if you run one outside the cluster. |

The Helm override file at `deploy/helm/umbrella/values-dev.yaml` mirrors those values. The top-level `backingServices` block documents the cluster hostnames for Postgres, NATS, and SeaweedFS, while each sub-chart override sets environment variables and OTLP endpoints (for example, `goosed-api.env.DB_DSN`). Update this file if you install the dependencies with different release names or credentials.

Ingress hosts used by the dev configuration:

* `api.goose.local`
* `boot.goose.local`
* `artifacts.goose.local`

Point each host at `127.0.0.1` in `/etc/hosts` when testing through the ingress controller on Docker Desktop.

## Rendering Kickstart & Unattend

1. **Expose the API locally**

   ```bash
   kubectl -n goose port-forward svc/goosed-api 18080:8080
   ```

   Leave this running in a separate terminal.

2. **Enroll a machine** – the API stores machine definitions. Use the sample profiles under `infra/machine-profiles/lab-a/rack-01/` as a starting point. The example below registers the first RHEL node.

   ```bash
   cat <<'JSON' > /tmp/rhel-machine.json
   {
     "mac": "00:11:22:aa:bb:cc",
     "serial": "LABA-RACK01-01",
     "profile": {
       "blueprint": "rhel/9/base",
       "workflow": "rhel-default",
       "hostname": "laba-r01-n01.goose.local",
       "packages": ["vim", "tmux", "git"],
       "kickstart": {
         "timezone": "UTC",
         "rootPasswordHash": "$6$rounds=4096$goosedlab$0cY9wj7v2uZB7Q2yEn8g9orL49HRvPxvXB1EZVZhG7T0ioigj8O4d2o2i0L1.BFQJ6xXzB/8C2Tkq5VdJtA1p."
       }
     }
   }
   JSON

   MACHINE_ID=$(curl -sSf -X POST http://127.0.0.1:18080/v1/machines \
     -H 'Content-Type: application/json' \
     --data @/tmp/rhel-machine.json | jq -r '.machine.id')
   ```

3. **Render Kickstart or Unattend**

   ```bash
   curl -sSf "http://127.0.0.1:18080/v1/render/kickstart?machine_id=${MACHINE_ID}" > kickstart.cfg
   curl -sSf "http://127.0.0.1:18080/v1/render/unattend?machine_id=${MACHINE_ID}" > unattend.xml
   ```

   Swap in the Windows profile from `infra/machine-profiles/lab-a/rack-01/10-mac-00aa11bb22cc-windows.yaml` when you need an Unattend payload that includes driver locations and post-install commands.

4. **Clean up** – stop the port-forward (`Ctrl+C`). The issued token is single-use; the next render call will mint a new one automatically.

## Deploying the Stack

**Umbrella chart**

```bash
helm dependency build deploy/helm/umbrella
helm upgrade --install goose deploy/helm/umbrella \
  --namespace goose \
  -f deploy/helm/umbrella/values-dev.yaml
```

Layer extra `-f` files or `--set key=value` flags when you need to override endpoints, credentials, or image tags. Every service chart lives under `deploy/helm/<service>/` and can be upgraded independently during development:

```bash
helm upgrade --install goose-api deploy/helm/goosed-api \
  --namespace goose \
  --set image.tag=$(git rev-parse --short HEAD)
```

## PXE Boot: Dev vs Lab

* **Dev (Docker Desktop K8s)**
  * Docker Desktop does not expose raw L2 networking, so rely on **HTTPBoot/iPXE** with static host mappings.
  * Run DHCP/TFTP in a lightweight VM (dnsmasq) outside the cluster or hand out iPXE USB sticks for quick tests.
  * Port-forward `bootd` when you need to exercise the menu (`kubectl -n goose port-forward svc/goosed-bootd 18081:8080`). Branding assets live under `infra/branding/` and hot-reload without redeploying the chart.
  * Keep large artifacts (ISOs/WIMs) in SeaweedFS via the `goose-seaweedfs` release; the ingress rules already forward Range requests to support resumable downloads.

* **Lab / Air-gapped**
  * Deploy `bootd` on hardware that sits directly on the provisioning VLAN and enable **ProxyDHCP + TFTP** if legacy BIOS machines still exist.
  * Mirror container images, RHEL repos, and Windows drivers using `goosectl bundles` so the lab never needs internet access.
  * Terminate TLS at the edge (ingress controller or metal load balancer) and ensure `bootd` trusts the internal CA when chaining to the API.
  * When spanning racks, place SeaweedFS volumes close to the PXE network to avoid saturating the control-plane uplinks with large ISO fetches.

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

Export an **age** secret key (`AGE-SECRET-KEY-...`) before building. To avoid placing the private key on the import host, derive a verifier public key (base64 Ed25519) once and store it securely:

```bash
export AGE_SECRET_KEY=$(cat ~/.config/goosed/age.key)
export AGE_PUBLIC_KEY=$(go run - <<'EOF'
package main

import (
        "fmt"

        "goosed/services/bundler"
)

func main() {
        signer, err := bundler.NewSignerFromEnv()
        if err != nil {
                panic(err)
        }
        fmt.Println(signer.PublicKeyBase64())
}
EOF
)
```

`AGE_SECRET_KEY` is required for signing; either `AGE_SECRET_KEY` **or** `AGE_PUBLIC_KEY` must be present when importing.

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
export AGE_PUBLIC_KEY=<base64-ed25519-from-above>   # or reuse AGE_SECRET_KEY
export S3_ENDPOINT=https://seaweedfs.example.local:8333
export S3_ACCESS_KEY=...
export S3_SECRET_KEY=...
export S3_REGION=us-east-1
export S3_DISABLE_TLS=false

go run ./services/bundler/cmd/goosectl \
  bundles import \
  --file ./bundle-20251104.tar.zst \
  --api https://api.goose.local
```

**Offline flow**

1. On a connected workstation, run the build command to generate the `bundle-*.tar.zst` archive.
2. Transfer the bundle and the `AGE_PUBLIC_KEY` value to the air-gapped environment (keep the secret key offline).
3. Configure the air-gapped host with S3 credentials, `AGE_PUBLIC_KEY`, and the API URL, then run the import command.
4. `goosectl` verifies the signed manifest, uploads each object via the S3 API with checksums, and registers it through `POST /v1/artifacts` in register-only mode.
5. Confirm availability with `GET /v1/artifacts` or the dashboard before scheduling installs.

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
