# Deploying goose'd in Kubernetes

Use this guide when you need to redeploy individual services, override Helm values, or upgrade the full stack after following the [getting started steps](getting-started.md).

## Umbrella chart deployment

The umbrella chart coordinates all services and their shared dependencies. Rebuild the chart dependencies and deploy it into the `goose` namespace:

```bash
helm dependency build deploy/helm/umbrella
helm upgrade --install goose deploy/helm/umbrella \
  --namespace goose \
  -f deploy/helm/umbrella/values-dev.yaml
```

Layer extra `-f` files or `--set key=value` flags when you need to override endpoints, credentials, or image tags.

## Deploying a single service chart

Every service chart lives under `deploy/helm/<service>/` and can be upgraded independently during development. Example for the API service:

```bash
helm upgrade --install goose-api deploy/helm/goosed-api \
  --namespace goose \
  --set image.tag=$(git rev-parse --short HEAD)
```

Swap the chart path and overrides for the service you are targeting.
