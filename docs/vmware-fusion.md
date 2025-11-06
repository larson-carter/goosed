# PXE Booting VMware Fusion Guests

The `pxe-stack` helpers ship with the default deployment, bringing DHCP, TFTP, and a PXE-friendly HTTP endpoint into your cluster so VMware Fusion VMs can boot directly from goose'd.

1. **Confirm the stack in Kubernetes**
   * Deploy the platform via [getting-started.md](getting-started.md) to install the PXE helpers alongside the core services, then verify the pod is running:

     ```bash
     kubectl -n goose get pods -l app.kubernetes.io/name=goosed-pxe-stack
     ```

   * The chart publishes three ports: DHCP (`67/udp`), TFTP (`69/udp`), and HTTP (`8080/tcp`). Expose them with a `LoadBalancer` or `NodePort` service and ensure the PXE VLAN can reach the selected nodes.

2. **Bridge your VMware Fusion VM to the PXE network**
   * Create a new VM with no installation media and set its network adapter to **Bridged (Autodetect)** so it shares the same layer-2 segment as the `pxe-stack` service.
   * Power on the VM and press **Esc** (UEFI) or **F12** (BIOS) to pick **EFI Network**/**Network Boot**. You should see the VM obtain an address from the DHCP handler, download `undionly.kpxe` from TFTP, then chainload into `bootd`.

3. **Serve installation media from SeaweedFS**
   * Use the helper script at the repository root to push the ISO into SeaweedFS. Export your SeaweedFS credentials (defaults shown below), then point the script at the downloaded Rocky Linux image:

```bash
   export S3_ENDPOINT=http://localhost:8333
   export S3_BUCKET=goosed-artifacts
   export S3_ACCESS_KEY=goosed
   export S3_SECRET_KEY=goosedsecret
   export ISO_S3_KEY=artifacts/rocky/9/Rocky-9.4-x86_64-minimal.iso
   ./copy-iso.sh ./Rocky-9.4-x86_64-minimal.iso
```

   * The script accepts the ISO path as an argument or, if omitted, prompts for it interactively. By default it places the object at `artifacts/rocky/9/<iso-filename>` inside the bucket and will use the `aws` CLI when available. When `aws` is missing it will reuse an existing `mc` alias or download a temporary copy and seed a new alias with the credentials above. If `mc` reports a signature or credential mismatch, the helper automatically reconfigures the alias across `S3v2`, `S3v4`, or any values provided via `MC_API` (comma- or space-separated) until the upload succeeds. Override the destination with `ISO_S3_KEY`, and set `MC_ALIAS` when you need to target a non-default alias name.
   * Upload Rocky Linux media into the artifacts bucket exactly where the blueprint expects it. For Rocky 9 the canonical object key is `artifacts/rocky/9/Rocky-9.4-x86_64-minimal.iso`; kernels and initrds live alongside it under `artifacts/rocky/9/`.
   * Prefer the script for routine uploads, but you can still follow the [SeaweedFS ISO upload walkthrough](seaweedfs-iso-upload.md) for manual steps or additional background. `goosectl bundles` can package the ISO for offline import as well.

Once the VM is online, the normal Kickstart workflow takes over: `bootd` hands iPXE a tokenised URL, the API renders a host-specific Kickstart, and the guest installs from the ISO hosted in SeaweedFS. Continue with the [provisioning flows guide](provisioning-flows.md) for end-to-end context.
