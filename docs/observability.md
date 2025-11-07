# Observability with Grafana

The `goosed-observability` Helm chart deploys Prometheus, Loki, Tempo, and Grafana as a
self-contained stack. Grafana is pre-loaded with the **Goosed Observability Overview**
dashboards and pre-provisioned data sources so you can review metrics, logs, and traces
immediately after installation.

## Port-forward Grafana

Expose Grafana locally from the `goose` namespace:

```bash
kubectl -n goose port-forward svc/goosed-observability-grafana 3000:3000
```

Open http://localhost:3000 in a browser. Default credentials are set in
`deploy/helm/goosed-observability/values.yaml` (the development values ship with
`admin` / `admin`).

## Default home dashboard

After signing in you land on the **Goosed Observability Overview** dashboard at
`/d/goosed-overview/goosed-observability-overview`. Use the `Service` dropdown at the
top to focus on a specific workload or keep `All` selected for a global view.

Panels provided out of the box include:

* **HTTP Request Latency** (p50/p95/p99)
* **Request Throughput** (requests/second)
* **5xx Error Rate**
* **Logs** (live Loki stream scoped by the service filter)
* **Recent Traces** (Tempo search filtered by the service label)

## Exploring metrics, logs, and traces directly

Grafana's Explore view also ships ready-to-use data sources:

* **Prometheus** (`prometheus` uid) for ad-hoc metrics queries.
* **Loki** (`loki` uid) for raw log exploration. Select the Loki data source and the
  `service` label is already available to match the dashboard filter.
* **Tempo** (`tempo` uid) for distributed tracing. You can pivot from logs to traces
  using the built-in **View Trace** action on log lines that carry `trace_id` metadata.

Because Loki is provisioned as part of the chart, you do not need to perform any manual
configuration to access logs—the data source and dashboard panel are active on first
login.
