# Getting Started with goose'd

These steps walk through the minimal tooling you need on a workstation and how to bring up the platform on Docker Desktop's Kubernetes cluster.

## Prerequisites

Make sure the following tools and services are available before you begin:

* **Docker Desktop 4.x with Kubernetes enabled** (or a single-node alternative such as kind/minikube).
* **kubectl 1.29+** with a context pointed at your local cluster.
* **Helm 3.13+** for chart management.
* **Go 1.25+**, **make**, **git**, **bash**, **curl**, **jq**, and the Docker Compose plugin (used by `setup-env.sh`).
* **golangci-lint v1.61.0+** so `make lint` matches CI behaviour.
* Optional: **VS Code + Dev Containers** (the repo ships `.devcontainer/`).
* Outbound internet access the first time you pull container images and Helm charts.

## Quickstart (Docker Desktop Kubernetes)

1. **Clone the repository**

   ```bash
   git clone <your_repo> goosed
   cd goosed
   ```

2. **Bootstrap local tooling** – pulls development containers, generates `.env.development`, and checks Postgres/NATS/SeaweedFS reachability on the host.

   ```bash
   ./setup-env.sh
   source .env.development
   ```

3. **Prepare Go modules**

   ```bash
   make tidy
   ```

4. **Build containers**

   ```bash
   make build
   ```

5. **Install backing services into Kubernetes** – run once per cluster. The commands below create the `goose` namespace and install Postgres, NATS JetStream, and SeaweedFS with credentials that match `values-dev.yaml`.

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

6. **Deploy goose'd**

   ```bash
   helm dependency build deploy/helm/umbrella
   helm upgrade --install goose deploy/helm/umbrella \
     --namespace goose \
     -f deploy/helm/umbrella/values-dev.yaml
   ```

7. **Verify pods** – everything should settle into `Running`/`Completed` within a couple of minutes.

   ```bash
   kubectl -n goose get pods
   ```

Continue with the [deployment guide](deploying.md) if you want to customise the Helm release or upgrade services independently.
