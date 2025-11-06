# RHEL/Rocky and Windows Provisioning Flows

Use these notes alongside the [macOS hypervisor walkthrough](macos-hypervisors.md) or your lab deployment to understand what each stage of provisioning does.

## RHEL & Rocky (Kickstart)

1. PXE → iPXE → `GET /v1/boot/ipxe?mac=...` (API) → dynamic Kickstart URL.
2. Kickstart renders with repository mirrors, partitioning, and users, then `%post` installs **agent-rhel**.
3. First boot: the agent runs packages and hardening tasks, posts **facts**, and the orchestrator marks the **run** complete.

**Kickstart template:** `pkg/render/templates/kickstart.tmpl`

Rocky Linux shares the same Kickstart flow as RHEL. Use `infra/blueprints/rocky/9/base/blueprint.yaml` and `infra/workflows/rocky-default.yaml` together with a machine profile such as `infra/machine-profiles/lab-a/rack-01/03-mac-001122ccddee.yaml` when testing against the Rocky Linux ISO in a lab or local VM.

## Windows (WinPE/Unattend)

1. iPXE + **wimboot** loads WinPE (HTTP).
2. `provision.ps1` fetches Unattend, runs `DISM /Apply-Image`, injects drivers, and configures **agent-windows**.
3. The agent posts **facts** and the orchestrator completes the run.

**Unattend template:** `pkg/render/templates/unattend.xml.tmpl`

**WinPE script:** `services/agents/windows/provision.ps1`
