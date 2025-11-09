export type MachineStatus =
  | "ready"
  | "provisioning"
  | "error"
  | "offline"
  | "maintenance"
  | "unknown";

export type RunStatus = "running" | "succeeded" | "failed" | "unknown";

export type StatusCounts = Record<MachineStatus, number>;

export interface MachinesPayload {
  machines: MachineListItem[];
}

export interface MachineListItem {
  machine: APIMachine;
  status?: string | null;
  latest_fact?: MachineFact | null;
  recent_runs?: APIRun[] | null;
}

export interface APIMachine {
  id: string;
  mac: string;
  serial?: string;
  profile?: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

export interface MachineFact {
  id: string;
  snapshot?: Record<string, unknown> | null;
  created_at: string;
}

export interface APIRun {
  id: string;
  machine_id: string;
  blueprint_id: string;
  status: string;
  started_at?: string | null;
  finished_at?: string | null;
  logs?: string | null;
}

export interface MachineRecord {
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

export interface MachineNetwork {
  interface: string;
  mac: string;
  ip?: string;
  vlan?: string;
}

export interface MachineRun {
  id: string;
  blueprint?: string;
  status: RunStatus;
  startedAt?: string;
  finishedAt?: string;
}

export interface MachineFactEntry {
  label: string;
  value: string;
  updatedAt?: string;
}
