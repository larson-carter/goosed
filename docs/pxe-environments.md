# PXE Boot Strategies: Development vs Lab

Choose the workflow that matches your environment when exercising the PXE stack.

## Development (Docker Desktop Kubernetes)

* Docker Desktop does not expose raw layer-2 networking, so rely on **HTTPBoot/iPXE** with static host mappings.
* Run DHCP/TFTP in a lightweight VM (for example `dnsmasq`) outside the cluster or hand out iPXE USB sticks for quick tests.
* Port-forward `bootd` when you need to exercise the menu:

  ```bash
  kubectl -n goose port-forward svc/goosed-bootd 18081:8080
  ```

  Branding assets live under `infra/branding/` and hot-reload without redeploying the chart.

* Keep large artifacts (ISOs/WIMs) in SeaweedFS via the `goose-seaweedfs` release; the ingress rules already forward Range requests to support resumable downloads.

## Lab / Air-gapped

* Deploy `bootd` on hardware that sits directly on the provisioning VLAN and enable **ProxyDHCP + TFTP** if legacy BIOS machines still exist.
* Mirror container images, RHEL repositories, and Windows drivers using `goosectl bundles` so the lab never needs internet access.
* Terminate TLS at the edge (ingress controller or metal load balancer) and ensure `bootd` trusts the internal CA when chaining to the API.
* When spanning racks, place SeaweedFS volumes close to the PXE network to avoid saturating the control-plane uplinks with large ISO fetches.

> **UEFI Secure Boot:** sign iPXE or use a trusted shim if you need Secure Boot enabled.
