export interface BlueprintRecord {
  id: string;
  name: string;
  os: string;
  version: string;
  data: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface BlueprintsResponse {
  blueprints: BlueprintRecord[];
}

export interface BlueprintResponse {
  blueprint: BlueprintRecord;
}
