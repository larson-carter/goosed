# PXE Booting VMware Fusion Guests

Use the optional `pxe-stack` helpers to bring DHCP, TFTP, and a PXE-friendly HTTP endpoint into your cluster so VMware Fusion VMs can boot directly from goose'd.

1. **Enable the stack in Kubernetes**
   * This guide assumes you've already deployed the platform via [getting-started.md](getting-started.md); the command below upgrades that existing release to enable PXE support.
   * Toggle the Helm values to deploy the helpers alongside the rest of the platform:

     ```bash
     helm upgrade --install goose ./deploy/helm/umbrella \
       --namespace goose \
       --create-namespace \
       --set pxeStack.enabled=true
     ```

   * The chart publishes three ports: DHCP (`67/udp`), TFTP (`69/udp`), and HTTP (`8080/tcp`). Expose them with a `LoadBalancer` or `NodePort` service and ensure the PXE VLAN can reach the selected nodes.

2. **Bridge your VMware Fusion VM to the PXE network**
   * Create a new VM with no installation media and set its network adapter to **Bridged (Autodetect)** so it shares the same layer-2 segment as the `pxe-stack` service.
   * Power on the VM and press **Esc** (UEFI) or **F12** (BIOS) to pick **EFI Network**/**Network Boot**. You should see the VM obtain an address from the DHCP handler, download `undionly.kpxe` from TFTP, then chainload into `bootd`.

3. **Serve installation media from SeaweedFS**
   * Upload Rocky Linux media into the artifacts bucket exactly where the blueprint expects it. For Rocky 9 the canonical object key is `artifacts/rocky/9/Rocky-9.4-x86_64-minimal.iso`; kernels and initrds live alongside it under `artifacts/rocky/9/`.
   * Follow the [SeaweedFS ISO upload walkthrough](seaweedfs-iso-upload.md) to place the ISO in SeaweedFS using either the AWS CLI or `mc`.
   * `goosectl bundles` can package the ISO for offline import, or you can `mc cp`/`aws s3 cp` directly against the SeaweedFS S3 endpoint exposed by the `goose-seaweedfs` chart.

Once the VM is online, the normal Kickstart workflow takes over: `bootd` hands iPXE a tokenised URL, the API renders a host-specific Kickstart, and the guest installs from the ISO hosted in SeaweedFS. Continue with the [provisioning flows guide](provisioning-flows.md) for end-to-end context.
