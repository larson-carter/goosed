import {
  type ComponentType,
  type SVGProps,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  CalendarClock,
  CheckCircle2,
  ChevronRight,
  Circle,
  Clock4,
  Cpu,
  HardDrive,
  LaptopMinimal,
  MapPin,
  Network,
  RefreshCcw,
  Search,
  ServerCog,
  Sparkles,
  Tag,
  TimerReset,
} from "lucide-react";

const API_BASE_URL = (() => {
  const raw = import.meta.env.VITE_API_BASE_URL;
  if (typeof raw === "string" && raw.trim().length > 0) {
    return raw.replace(/\/$/, "");
  }
  return "/api";
})();

type MachineStatus =
  | "ready"
  | "provisioning"
  | "error"
  | "offline"
  | "maintenance"
  | "unknown";

type RunStatus = "running" | "succeeded" | "failed" | "unknown";

type StatusCounts = Record<MachineStatus, number>;

const STATUS_META: Record<MachineStatus, { label: string; badge: string; dot: string }> = {
  ready: {
    label: "Ready",
    badge: "border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
    dot: "bg-emerald-500",
  },
  provisioning: {
    label: "Provisioning",
    badge: "border-sky-500/40 bg-sky-500/10 text-sky-600 dark:text-sky-300",
    dot: "bg-sky-500",
  },
  error: {
    label: "Needs attention",
    badge: "border-rose-500/50 bg-rose-500/10 text-rose-600 dark:text-rose-300",
    dot: "bg-rose-500",
  },
  offline: {
    label: "Offline",
    badge: "border-zinc-400/60 bg-zinc-400/10 text-zinc-600 dark:text-zinc-300",
    dot: "bg-zinc-400",
  },
  maintenance: {
    label: "Maintenance",
    badge: "border-amber-500/50 bg-amber-500/10 text-amber-600 dark:text-amber-300",
    dot: "bg-amber-500",
  },
  unknown: {
    label: "Unknown",
    badge: "border-muted-foreground/40 bg-muted/30 text-muted-foreground",
    dot: "bg-muted-foreground",
  },
};

const RUN_META: Record<RunStatus, { label: string; badge: string; dot: string }> = {
  succeeded: {
    label: "Succeeded",
    badge: "border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
    dot: "bg-emerald-500",
  },
  running: {
    label: "Running",
    badge: "border-sky-500/40 bg-sky-500/10 text-sky-600 dark:text-sky-300",
    dot: "bg-sky-500",
  },
  failed: {
    label: "Failed",
    badge: "border-rose-500/50 bg-rose-500/10 text-rose-600 dark:text-rose-300",
    dot: "bg-rose-500",
  },
  unknown: {
    label: "Unknown",
    badge: "border-muted-foreground/40 bg-muted/30 text-muted-foreground",
    dot: "bg-muted-foreground",
  },
};

interface MachinesPayload {
  machines: MachineListItem[];
}

interface MachineListItem {
  machine: APIMachine;
  status?: string | null;
  latest_fact?: MachineFact | null;
  recent_runs?: APIRun[] | null;
}

interface APIMachine {
  id: string;
  mac: string;
  serial?: string;
  profile?: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

interface MachineFact {
  id: string;
  snapshot?: Record<string, unknown> | null;
  created_at: string;
}

interface APIRun {
  id: string;
  machine_id: string;
  blueprint_id: string;
  status: string;
  started_at?: string | null;
  finished_at?: string | null;
  logs?: string | null;
}

interface MachineRecord {
  id: string;
  mac: string;
  serial?: string;
  hostname: string;
  displayName?: string;
  status: MachineStatus;
  ip?: string;
  os?: string;
  blueprint?: string;
  site?: string;
  rack?: string;
  position?: string;
  owner?: string;
  notes?: string;
  tags: string[];
  lastCheckIn?: string;
  uptimeHours?: number;
  hardware: {
    cpu?: string;
    memory?: string;
    storage?: string;
    bmc?: string;
  };
  networks: MachineNetwork[];
  runs: MachineRun[];
  facts: MachineFactEntry[];
}

interface MachineNetwork {
  interface: string;
  mac: string;
  ip?: string;
  vlan?: string;
}

interface MachineRun {
  id: string;
  blueprint?: string;
  status: RunStatus;
  startedAt?: string;
  finishedAt?: string;
}

interface MachineFactEntry {
  label: string;
  value: string;
  updatedAt?: string;
}

function useMachinesQuery() {
  return useQuery<MachinesPayload, Error>({
    queryKey: ["machines"],
    queryFn: async () => {
      const response = await fetch(`${API_BASE_URL}/v1/machines`, {
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error(`Request failed with status ${response.status}`);
      }
      const payload = (await response.json()) as MachinesPayload;
      if (!payload || !Array.isArray(payload.machines)) {
        return { machines: [] };
      }
      return payload;
    },
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

function formatRelativeTime(value?: string | null) {
  if (!value) {
    return "Unknown";
  }
  const date = new Date(value);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();

  if (!Number.isFinite(diffMs) || diffMs < 0) {
    return "Unknown";
  }

  const diffMinutes = Math.floor(diffMs / 60000);
  if (diffMinutes < 1) {
    return "just now";
  }
  if (diffMinutes < 60) {
    return `${diffMinutes}m ago`;
  }
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) {
    return `${diffHours}h ago`;
  }
  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) {
    return `${diffDays}d ago`;
  }
  const diffWeeks = Math.floor(diffDays / 7);
  if (diffWeeks < 5) {
    return `${diffWeeks}w ago`;
  }
  const diffMonths = Math.floor(diffDays / 30);
  if (diffMonths < 12) {
    return `${diffMonths}mo ago`;
  }
  const diffYears = Math.floor(diffDays / 365);
  return `${diffYears}y ago`;
}

function formatAbsolute(value?: string | null) {
  if (!value) {
    return "Unknown";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Unknown";
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function formatUptime(hours?: number) {
  if (!Number.isFinite(hours) || hours === undefined || hours <= 0) {
    return "Unknown";
  }
  if (hours < 24) {
    return `${Math.round(hours)}h`;
  }
  if (hours < 24 * 14) {
    return `${Math.round(hours / 24)}d`;
  }
  return `${Math.round(hours / (24 * 7))}w`;
}

export function MachinesPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<MachineStatus | "all">("all");
  const [siteFilter, setSiteFilter] = useState<string>("all");
  const [selectedMachineId, setSelectedMachineId] = useState<string>("");

  const { data, isLoading, isError, error, refetch, isRefetching } =
    useMachinesQuery();

  const machines = useMemo(() => {
    if (!data?.machines) {
      return [] as MachineRecord[];
    }
    return data.machines.map(normaliseMachineRecord);
  }, [data]);

  useEffect(() => {
    if (machines.length === 0) {
      setSelectedMachineId("");
      return;
    }
    if (!machines.some((item) => item.id === selectedMachineId)) {
      setSelectedMachineId(machines[0].id);
    }
  }, [machines, selectedMachineId]);

  const statusCounts = useMemo(() => computeStatusCounts(machines), [machines]);
  const sites = useMemo(() => computeSiteOptions(machines), [machines]);

  const filteredMachines = useMemo(() => {
    return machines.filter((machine) => {
      if (statusFilter !== "all" && machine.status !== statusFilter) {
        return false;
      }
      if (siteFilter !== "all" && machine.site !== siteFilter) {
        return false;
      }
      if (!searchTerm.trim()) {
        return true;
      }
      const query = searchTerm.trim().toLowerCase();
      const tokens = [
        machine.hostname,
        machine.displayName,
        machine.serial,
        machine.os,
        machine.blueprint,
        machine.mac,
        machine.ip,
        machine.owner,
        machine.site,
        machine.rack,
        ...(machine.tags ?? []),
      ]
        .filter((value): value is string => typeof value === "string" && value.trim().length > 0)
        .join(" ")
        .toLowerCase();
      return tokens.includes(query);
    });
  }, [machines, searchTerm, siteFilter, statusFilter]);

  useEffect(() => {
    if (!filteredMachines.some((item) => item.id === selectedMachineId)) {
      setSelectedMachineId(filteredMachines[0]?.id ?? "");
    }
  }, [filteredMachines, selectedMachineId]);

  const selectedMachine = useMemo<MachineRecord | undefined>(() => {
    return machines.find((item) => item.id === selectedMachineId);
  }, [machines, selectedMachineId]);

  const filtersActive =
    statusFilter !== "all" || siteFilter !== "all" || searchTerm.trim().length > 0;

  const summaryCards = [
    {
      label: "Ready for workloads",
      value: statusCounts.ready,
      icon: CheckCircle2,
      helper: "Healthy & reported",
      change: `${statusCounts.ready} total`,
      iconStyles: "text-emerald-500",
    },
    {
      label: "In provisioning",
      value: statusCounts.provisioning,
      icon: Sparkles,
      helper: "Automation active",
      change: `${statusCounts.provisioning} running now`,
      iconStyles: "text-sky-500",
    },
    {
      label: "Needs attention",
      value: statusCounts.error,
      icon: AlertTriangle,
      helper: "Blocking issues",
      change: statusCounts.error ? `${statusCounts.error} flagged` : "All clear",
      iconStyles: "text-rose-500",
    },
    {
      label: "Offline or maintenance",
      value: statusCounts.offline + statusCounts.maintenance + statusCounts.unknown,
      icon: TimerReset,
      helper: "Awaiting action",
      change:
        statusCounts.unknown > 0
          ? `${statusCounts.unknown} unknown`
          : `${statusCounts.offline} offline`,
      iconStyles: "text-amber-500",
    },
  ];

  return (
    <section className="space-y-6">
      <div className="flex flex-col gap-2">
        <h2 className="text-2xl font-semibold">Machines</h2>
        <p className="text-sm text-muted-foreground">
          Manage goose’d infrastructure, track provisioning runs, and inspect the latest
          inventory snapshots.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {summaryCards.map((card) => (
          <div
            key={card.label}
            className="rounded-lg border bg-card p-4 shadow-sm transition-colors hover:border-primary/40"
          >
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-xs uppercase tracking-wide text-muted-foreground">
                  {card.label}
                </p>
                <p className="mt-2 text-2xl font-semibold">{card.value}</p>
              </div>
              <card.icon className={`h-9 w-9 ${card.iconStyles}`} aria-hidden />
            </div>
            <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
              <span>{card.helper}</span>
              <span>{card.change}</span>
            </div>
          </div>
        ))}
      </div>

      <div className="rounded-lg border bg-card p-4 shadow-sm">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div className="relative w-full md:max-w-xs">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input
              type="search"
              placeholder="Search by hostname, MAC, owner, blueprint…"
              className="w-full rounded-md border px-9 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              value={searchTerm}
              onChange={(event) => setSearchTerm(event.target.value)}
            />
          </div>
          <div className="flex flex-wrap items-center gap-3 text-sm">
            <label className="flex items-center gap-2">
              <span className="text-muted-foreground">Status</span>
              <select
                className="rounded-md border bg-background px-2.5 py-2 focus:border-primary focus:outline-none"
                value={statusFilter}
                onChange={(event) => setStatusFilter(event.target.value as MachineStatus | "all")}
              >
                <option value="all">All</option>
                <option value="ready">Ready</option>
                <option value="provisioning">Provisioning</option>
                <option value="error">Needs attention</option>
                <option value="offline">Offline</option>
                <option value="maintenance">Maintenance</option>
                <option value="unknown">Unknown</option>
              </select>
            </label>
            <label className="flex items-center gap-2">
              <span className="text-muted-foreground">Site</span>
              <select
                className="rounded-md border bg-background px-2.5 py-2 focus:border-primary focus:outline-none"
                value={siteFilter}
                onChange={(event) => setSiteFilter(event.target.value)}
              >
                <option value="all">All</option>
                {sites.map((site) => (
                  <option key={site} value={site}>
                    {site}
                  </option>
                ))}
              </select>
            </label>
            {(filtersActive || isRefetching) && (
              <button
                type="button"
                className="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-xs font-medium uppercase tracking-wide text-muted-foreground transition-colors hover:border-primary/40 hover:text-primary"
                onClick={() => {
                  if (filtersActive) {
                    setSearchTerm("");
                    setSiteFilter("all");
                    setStatusFilter("all");
                  } else {
                    refetch();
                  }
                }}
              >
                {filtersActive ? "Clear filters" : "Refreshing"}
              </button>
            )}
            {isError && (
              <button
                type="button"
                className="inline-flex items-center gap-2 rounded-md border border-destructive/50 px-3 py-2 text-xs font-semibold text-destructive transition-colors hover:border-destructive hover:text-destructive"
                onClick={() => refetch()}
              >
                <RefreshCcw className="h-3.5 w-3.5" aria-hidden />
                Retry
              </button>
            )}
          </div>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-[1fr_340px] xl:grid-cols-[1fr_380px]">
        <div className="rounded-lg border bg-card shadow-sm">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[720px] table-fixed text-sm">
              <thead className="bg-muted/50 text-xs uppercase tracking-wide text-muted-foreground">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold">Hostname</th>
                  <th className="px-4 py-3 text-left font-semibold">Status</th>
                  <th className="px-4 py-3 text-left font-semibold">Network</th>
                  <th className="px-4 py-3 text-left font-semibold">Blueprint</th>
                  <th className="px-4 py-3 text-left font-semibold">Location</th>
                  <th className="px-4 py-3 text-left font-semibold">Last check-in</th>
                  <th className="px-4 py-3 text-left font-semibold">Tags</th>
                </tr>
              </thead>
              <tbody>
                {isLoading && (
                  <tr>
                    <td className="px-4 py-12 text-center text-sm text-muted-foreground" colSpan={7}>
                      Loading machines…
                    </td>
                  </tr>
                )}
                {isError && !isLoading && machines.length === 0 && (
                  <tr>
                    <td className="px-4 py-12 text-center text-sm text-muted-foreground" colSpan={7}>
                      Failed to load machines: {error?.message}
                    </td>
                  </tr>
                )}
                {!isLoading && !isError && filteredMachines.length === 0 && (
                  <tr>
                    <td className="px-4 py-12 text-center text-sm text-muted-foreground" colSpan={7}>
                      No machines match the current filters.
                    </td>
                  </tr>
                )}
                {filteredMachines.map((machine) => {
                  const isSelected = selectedMachineId === machine.id;
                  return (
                    <tr
                      key={machine.id}
                      className={`cursor-pointer border-t transition-colors ${
                        isSelected ? "bg-primary/5" : "hover:bg-muted/40"
                      }`}
                      onClick={() => setSelectedMachineId(machine.id)}
                    >
                      <td className="px-4 py-3 align-top font-medium">
                        <div className="flex items-center gap-2">
                          <ServerCog className="h-4 w-4 text-muted-foreground" aria-hidden />
                          <div>
                            <div>{machine.hostname}</div>
                            <div className="text-xs text-muted-foreground">
                              {(machine.serial && machine.serial.trim()) || machine.mac.toUpperCase()}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-3 align-top">
                        <StatusBadge status={machine.status} />
                      </td>
                      <td className="px-4 py-3 align-top">
                        <div className="flex flex-col gap-1">
                          <span className="font-medium">{machine.ip ?? "—"}</span>
                          <span className="text-xs text-muted-foreground">
                            {machine.mac.toUpperCase()}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3 align-top">
                        <div className="flex flex-col gap-1">
                          <span>{machine.os ?? machine.blueprint ?? "Unknown"}</span>
                          {machine.blueprint && (
                            <span className="text-xs text-muted-foreground">{machine.blueprint}</span>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3 align-top">
                        <div className="flex flex-col gap-1">
                          <span>{machine.site ?? "Unknown"}</span>
                          <span className="text-xs text-muted-foreground">
                            {[machine.rack, machine.position].filter(Boolean).join(" • ") || "—"}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3 align-top text-xs text-muted-foreground">
                        {formatRelativeTime(machine.lastCheckIn)}
                      </td>
                      <td className="px-4 py-3 align-top">
                        <div className="flex flex-wrap gap-2">
                          {machine.tags.map((tag) => (
                            <span
                              key={tag}
                              className="inline-flex items-center gap-1 rounded-full border border-muted-foreground/30 bg-muted/30 px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground"
                            >
                              <Tag className="h-3 w-3" aria-hidden />
                              {tag}
                            </span>
                          ))}
                          {machine.tags.length === 0 && <span className="text-xs text-muted-foreground">—</span>}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        <div className="rounded-lg border bg-card shadow-sm">
          {selectedMachine ? (
            <div className="flex h-full flex-col gap-5 p-6">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h3 className="text-lg font-semibold">{selectedMachine.hostname}</h3>
                  <p className="text-sm text-muted-foreground">
                    {selectedMachine.displayName ?? selectedMachine.serial ?? selectedMachine.mac}
                  </p>
                </div>
                <StatusBadge status={selectedMachine.status} />
              </div>

              {selectedMachine.notes && (
                <div className="rounded-md border border-primary/30 bg-primary/10 p-3 text-xs text-primary">
                  {selectedMachine.notes}
                </div>
              )}

              <div className="grid grid-cols-1 gap-3 text-sm">
                <InfoRow
                  icon={MapPin}
                  label="Location"
                  primary={[selectedMachine.site, selectedMachine.rack]
                    .filter(Boolean)
                    .join(" • ") || "Unknown"}
                  secondary={selectedMachine.position ? `Position ${selectedMachine.position}` : undefined}
                />
                <InfoRow
                  icon={LaptopMinimal}
                  label="Operating system"
                  primary={selectedMachine.os ?? selectedMachine.blueprint ?? "Unknown"}
                  secondary={selectedMachine.blueprint ?? undefined}
                />
                <InfoRow
                  icon={Cpu}
                  label="Hardware"
                  primary={selectedMachine.hardware.cpu ?? "Unknown"}
                  secondary={[selectedMachine.hardware.memory, selectedMachine.hardware.storage]
                    .filter(Boolean)
                    .join(" • ") || undefined}
                />
                <InfoRow
                  icon={HardDrive}
                  label="Out-of-band"
                  primary={selectedMachine.hardware.bmc ?? "Not configured"}
                  secondary={`MAC ${selectedMachine.mac.toUpperCase()}`}
                />
                <InfoRow
                  icon={Clock4}
                  label="Uptime"
                  primary={formatUptime(selectedMachine.uptimeHours)}
                  secondary={`Last check-in ${formatRelativeTime(selectedMachine.lastCheckIn)}`}
                />
                <InfoRow
                  icon={CalendarClock}
                  label="Ownership"
                  primary={selectedMachine.owner ?? "Unassigned"}
                  secondary={selectedMachine.lastCheckIn ? `Updated ${formatAbsolute(selectedMachine.lastCheckIn)}` : undefined}
                />
              </div>

              <div>
                <SectionTitle icon={Network} title="Network interfaces" />
                <div className="mt-2 space-y-2 text-sm">
                  {selectedMachine.networks.length > 0 ? (
                    selectedMachine.networks.map((net) => (
                      <div
                        key={`${net.interface}-${net.mac}`}
                        className="flex items-start justify-between gap-3 rounded-md border bg-muted/30 px-3 py-2"
                      >
                        <div>
                          <p className="font-medium">{net.interface}</p>
                          <p className="text-xs text-muted-foreground">{net.mac.toUpperCase()}</p>
                        </div>
                        <div className="text-right text-xs text-muted-foreground">
                          <p>{net.ip ?? "No IP assigned"}</p>
                          {net.vlan && <p>VLAN {net.vlan}</p>}
                        </div>
                      </div>
                    ))
                  ) : (
                    <p className="text-xs text-muted-foreground">No interfaces reported.</p>
                  )}
                </div>
              </div>

              <div>
                <SectionTitle icon={ServerCog} title="Recent runs" />
                <div className="mt-2 space-y-2 text-sm">
                  {selectedMachine.runs.length > 0 ? (
                    selectedMachine.runs.map((run) => (
                      <div key={run.id} className="rounded-md border bg-muted/30 p-3">
                        <div className="flex items-center justify-between gap-3">
                          <div>
                            <p className="font-medium">{run.blueprint ?? "Unknown blueprint"}</p>
                            <p className="text-xs text-muted-foreground">
                              {run.finishedAt
                                ? `${formatAbsolute(run.startedAt)} → ${formatAbsolute(run.finishedAt)}`
                                : `Started ${formatAbsolute(run.startedAt)}`}
                            </p>
                          </div>
                          <RunBadge status={run.status} />
                        </div>
                      </div>
                    ))
                  ) : (
                    <p className="text-xs text-muted-foreground">No runs recorded yet.</p>
                  )}
                </div>
              </div>

              <div>
                <SectionTitle icon={Circle} title="Latest facts" />
                <ul className="mt-2 space-y-2 text-sm">
                  {selectedMachine.facts.length > 0 ? (
                    selectedMachine.facts.map((fact) => (
                      <li
                        key={`${fact.label}-${fact.updatedAt}`}
                        className="flex items-start justify-between gap-3 rounded-md border bg-muted/20 px-3 py-2"
                      >
                        <div>
                          <p className="font-medium">{fact.label}</p>
                          <p className="text-xs text-muted-foreground">{fact.value}</p>
                        </div>
                        <span className="text-xs text-muted-foreground">
                          Updated {formatRelativeTime(fact.updatedAt)}
                        </span>
                      </li>
                    ))
                  ) : (
                    <li className="text-xs text-muted-foreground">No agent facts reported yet.</li>
                  )}
                </ul>
              </div>
            </div>
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-3 p-6 text-center text-sm text-muted-foreground">
              <ChevronRight className="h-6 w-6" aria-hidden />
              <p>Select a machine from the inventory table to inspect its details.</p>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}

function StatusBadge({ status }: { status: MachineStatus }) {
  const meta = STATUS_META[status] ?? STATUS_META.unknown;
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-semibold ${meta.badge}`}
    >
      <span className={`h-2 w-2 rounded-full ${meta.dot}`} aria-hidden />
      {meta.label}
    </span>
  );
}

function RunBadge({ status }: { status: RunStatus }) {
  const meta = RUN_META[status] ?? RUN_META.unknown;
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium ${meta.badge}`}
    >
      <span className={`h-2 w-2 rounded-full ${meta.dot}`} aria-hidden />
      {meta.label}
    </span>
  );
}

function InfoRow({
  icon: Icon,
  label,
  primary,
  secondary,
}: {
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  label: string;
  primary?: string;
  secondary?: string;
}) {
  return (
    <div className="flex items-start gap-3 rounded-md border bg-muted/20 px-3 py-2">
      <Icon className="mt-0.5 h-4 w-4 text-muted-foreground" aria-hidden />
      <div>
        <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
        <p className="text-sm font-medium">{primary ?? "Unknown"}</p>
        {secondary && <p className="text-xs text-muted-foreground">{secondary}</p>}
      </div>
    </div>
  );
}

function SectionTitle({
  icon: Icon,
  title,
}: {
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  title: string;
}) {
  return (
    <div className="flex items-center gap-2 text-sm font-semibold">
      <Icon className="h-4 w-4 text-muted-foreground" aria-hidden />
      <span>{title}</span>
    </div>
  );
}
function normaliseMachineRecord(item: MachineListItem): MachineRecord {
  const machine = item.machine;
  const profile = asRecord(machine.profile);
  const metadata = asRecord(profile?.metadata);
  const spec = asRecord(profile?.spec);
  const profileDetails = asRecord(profile?.profile) ?? asRecord(spec?.profile);
  const machineSpec = asRecord(profile?.machine) ?? asRecord(spec?.machine);
  const metadataLabels = asRecord(metadata?.labels);
  const metadataAnnotations = asRecord(metadata?.annotations);

  const hostname = pickFirstString(
    stringFrom(profile, ["hostname"]),
    stringFrom(profileDetails, ["hostname"]),
    stringFrom(machineSpec, ["hostname"]),
    stringFrom(metadata, ["name"]),
    machine.serial,
    machine.mac,
  );

  const displayName = pickFirstString(
    stringFrom(profile, ["display_name"]),
    stringFrom(profile, ["displayName"]),
    stringFrom(metadata, ["name"]),
    machine.serial,
  );

  const blueprint = pickFirstString(
    stringFrom(profile, ["blueprint"]),
    stringFrom(spec, ["blueprint"]),
    stringFrom(profileDetails, ["blueprint"]),
    stringFrom(machineSpec, ["blueprint"]),
  );

  const os = pickFirstString(
    stringFrom(profile, ["os"]),
    stringFrom(profileDetails, ["os"]),
    stringFrom(profileDetails, ["operating_system"]),
    stringFrom(machineSpec, ["os"]),
  );

  const site = pickFirstString(
    stringFrom(profile, ["site"]),
    stringFrom(machineSpec, ["site"]),
    stringFrom(metadataLabels, ["site"]),
  );

  const rack = pickFirstString(
    stringFrom(profile, ["rack"]),
    stringFrom(machineSpec, ["rack"]),
    stringFrom(metadataLabels, ["rack"]),
  );

  const position = pickFirstString(
    stringFrom(profile, ["position"]),
    stringFrom(machineSpec, ["position"]),
    stringFrom(machineSpec, ["slot"]),
  );

  const owner = pickFirstString(
    stringFrom(profile, ["owner"]),
    stringFrom(profileDetails, ["owner"]),
    stringFrom(metadataAnnotations, ["owner"]),
  );

  const notes = pickFirstString(stringFrom(profile, ["notes"]), stringFrom(profileDetails, ["notes"]));

  const network =
    asRecord(profile?.network) ??
    asRecord(machineSpec?.network) ??
    asRecord(profileDetails?.network);

  const ip = pickFirstString(
    stringFrom(profile, ["ip"]),
    stringFrom(network, ["ipv4", "address"]),
    stringFrom(network, ["ip"]),
  );

  const hardware =
    asRecord(profile?.hardware) ??
    asRecord(machineSpec?.hardware) ??
    asRecord(profileDetails?.hardware);

  const memoryValue = pickFirstString(
    stringFrom(hardware, ["memory"]),
    formatNumberUnit(numberFrom(hardware, ["memory_gb"]), "GB"),
    formatNumberUnit(numberFrom(hardware, ["memoryGB"]), "GB"),
  );

  const storageValue = pickFirstString(
    stringFrom(hardware, ["storage"]),
    stringFrom(hardware, ["disk"]),
  );

  const uptimeHours = numberFrom(item.latest_fact?.snapshot ?? {}, ["uptime_hours"]);

  const tags = collectTags(profile, profileDetails, metadataLabels);

  const networks = extractNetworks(network);

  const runs = (item.recent_runs ?? [])
    .filter((run): run is APIRun => Boolean(run && run.id))
    .map((run) => ({
      id: run.id,
      blueprint: run.blueprint_id,
      status: toRunStatus(run.status),
      startedAt: run.started_at ?? undefined,
      finishedAt: run.finished_at ?? undefined,
    }));

  const facts = formatFactEntries(item.latest_fact);

  return {
    id: machine.id,
    mac: machine.mac,
    serial: machine.serial,
    hostname: hostname ?? machine.mac,
    displayName,
    status: toMachineStatus(item.status),
    ip,
    os,
    blueprint,
    site,
    rack,
    position,
    owner,
    notes,
    tags,
    lastCheckIn: item.latest_fact?.created_at ?? machine.updated_at,
    uptimeHours,
    hardware: {
      cpu: pickFirstString(stringFrom(hardware, ["cpu"]), stringFrom(hardware, ["model"])),
      memory: memoryValue,
      storage: storageValue,
      bmc: pickFirstString(stringFrom(hardware, ["bmc"]), stringFrom(hardware, ["ilo"])),
    },
    networks,
    runs,
    facts,
  };
}

function toMachineStatus(status?: string | null): MachineStatus {
  const value = status?.toLowerCase().trim();
  switch (value) {
    case "ready":
      return "ready";
    case "provisioning":
    case "provisioned":
    case "installing":
      return "provisioning";
    case "error":
    case "failed":
      return "error";
    case "maintenance":
      return "maintenance";
    case "offline":
      return "offline";
    default:
      return "unknown";
  }
}

function toRunStatus(status?: string | null): RunStatus {
  const value = status?.toLowerCase().trim();
  switch (value) {
    case "running":
      return "running";
    case "success":
    case "succeeded":
    case "completed":
      return "succeeded";
    case "failed":
    case "failure":
    case "error":
    case "errored":
      return "failed";
    default:
      return "unknown";
  }
}

function computeStatusCounts(records: MachineRecord[]): StatusCounts {
  const counts: StatusCounts = {
    ready: 0,
    provisioning: 0,
    error: 0,
    offline: 0,
    maintenance: 0,
    unknown: 0,
  };

  for (const record of records) {
    counts[record.status] = (counts[record.status] ?? 0) + 1;
  }

  return counts;
}

function computeSiteOptions(records: MachineRecord[]): string[] {
  const sites = new Set<string>();
  for (const record of records) {
    if (record.site) {
      sites.add(record.site);
    }
  }
  return Array.from(sites).sort((a, b) => a.localeCompare(b));
}

function collectTags(
  profile?: Record<string, unknown>,
  profileDetails?: Record<string, unknown>,
  metadataLabels?: Record<string, unknown>,
) {
  const tags = new Set<string>();

  const profileTags = valueFromPath(profile, ["tags"]);
  if (Array.isArray(profileTags)) {
    profileTags
      .map((tag) => (typeof tag === "string" ? tag.trim() : ""))
      .filter((tag) => tag.length > 0)
      .forEach((tag) => tags.add(tag));
  }

  const detailsTags = valueFromPath(profileDetails, ["tags"]);
  if (Array.isArray(detailsTags)) {
    detailsTags
      .map((tag) => (typeof tag === "string" ? tag.trim() : ""))
      .filter((tag) => tag.length > 0)
      .forEach((tag) => tags.add(tag));
  }

  if (metadataLabels) {
    for (const [key, value] of Object.entries(metadataLabels)) {
      if (typeof value === "string" && value.trim().length > 0) {
        tags.add(`${key}:${value.trim()}`);
      }
    }
  }

  return Array.from(tags).sort((a, b) => a.localeCompare(b));
}

function extractNetworks(source?: Record<string, unknown>): MachineNetwork[] {
  if (!source) {
    return [];
  }
  const networks: MachineNetwork[] = [];

  const entries = valueFromPath(source, ["interfaces"]);
  if (Array.isArray(entries)) {
    for (const entry of entries) {
      const record = asRecord(entry);
      if (!record) {
        continue;
      }
      const iface = stringFrom(record, ["name"]) ?? stringFrom(record, ["interface"]);
      const mac = stringFrom(record, ["mac"]);
      if (!iface || !mac) {
        continue;
      }
      networks.push({
        interface: iface,
        mac,
        ip: stringFrom(record, ["ip"]),
        vlan: stringFrom(record, ["vlan"]),
      });
    }
  }

  if (networks.length === 0) {
    const iface = stringFrom(source, ["interface"]);
    const mac = stringFrom(source, ["mac"]);
    if (iface && mac) {
      networks.push({
        interface: iface,
        mac,
        ip: stringFrom(source, ["ip"]),
        vlan: stringFrom(source, ["vlan"]),
      });
    }
  }

  return networks;
}

function formatFactEntries(fact?: MachineFact | null): MachineFactEntry[] {
  const snapshot = asRecord(fact?.snapshot);
  if (!snapshot) {
    return [];
  }
  return Object.entries(snapshot)
    .slice(0, 12)
    .map(([key, value]) => ({
      label: formatFactLabel(key),
      value: formatFactValue(value),
      updatedAt: fact?.created_at,
    }));
}

function formatFactLabel(value: string) {
  return value
    .replace(/[_\-]+/g, " ")
    .split(" ")
    .map((part) => (part ? part[0]?.toUpperCase() + part.slice(1) : ""))
    .join(" ");
}

function formatFactValue(value: unknown) {
  if (value === null || value === undefined) {
    return "—";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  try {
    return JSON.stringify(value);
  } catch (error) {
    return "(complex value)";
  }
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined;
}

function valueFromPath(source: Record<string, unknown> | undefined, path: string[]): unknown {
  let current: unknown = source;
  for (const segment of path) {
    if (!asRecord(current)) {
      return undefined;
    }
    current = (current as Record<string, unknown>)[segment];
  }
  return current;
}

function stringFrom(source: Record<string, unknown> | undefined, path: string[]): string | undefined {
  const value = valueFromPath(source, path);
  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  }
  return undefined;
}

function numberFrom(source: Record<string, unknown> | undefined, path: string[]): number | undefined {
  const value = valueFromPath(source, path);
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return undefined;
}

function pickFirstString(...candidates: (string | undefined)[]): string | undefined {
  for (const candidate of candidates) {
    if (candidate && candidate.trim().length > 0) {
      return candidate.trim();
    }
  }
  return undefined;
}

function formatNumberUnit(value: number | undefined, unit: string) {
  if (value === undefined || !Number.isFinite(value)) {
    return undefined;
  }
  return `${value} ${unit}`;
}
