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