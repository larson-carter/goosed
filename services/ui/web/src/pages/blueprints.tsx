import {
  type ComponentType,
  type SVGProps,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import {
  CheckCircle2,
  Edit,
  LoaderCircle,
  Plus,
  RefreshCcw,
  Trash2,
} from "lucide-react";
import type {
  BlueprintRecord,
  BlueprintsResponse,
} from "../types/blueprints";

const API_BASE_URL = (() => {
  const raw = import.meta.env.VITE_API_BASE_URL;
  if (typeof raw === "string" && raw.trim().length > 0) {
    return raw.replace(/\/$/, "");
  }
  return "/api";
})();

type BlueprintPayload = {
  name: string;
  os: string;
  version: string;
  data: Record<string, unknown>;
};

type BlueprintFormValues = {
  name: string;
  os: string;
  version: string;
  data: string;
};

type MutationIcon = ComponentType<SVGProps<SVGSVGElement>>;

const DEFAULT_FORM_VALUES: BlueprintFormValues = {
  name: "",
  os: "",
  version: "",
  data: "{}",
};

function formatTimestamp(value?: string) {
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

type BlueprintsQueryOptions = {
  onSuccess?: (data: BlueprintsResponse) => void;
};

function useBlueprintsQuery(options?: BlueprintsQueryOptions) {
  return useQuery<BlueprintsResponse, Error>({
    queryKey: ["blueprints"],
    queryFn: async () => {
      const response = await fetch(`${API_BASE_URL}/v1/blueprints`, {
        credentials: "include",
      });
      if (!response.ok) {
        throw await extractError(response);
      }
      const payload = (await response.json()) as BlueprintsResponse;
      if (!payload || !Array.isArray(payload.blueprints)) {
        return { blueprints: [] } satisfies BlueprintsResponse;
      }
      return payload;
    },
    staleTime: 30_000,
    refetchInterval: 60_000,
    ...options,
  });
}

async function extractError(response: Response) {
  try {
    const payload = (await response.json()) as { error?: string };
    if (payload?.error) {
      return new Error(payload.error);
    }
  } catch (error) {
    console.error("Failed to parse error payload", error);
  }
  return new Error(`Request failed with status ${response.status}`);
}

export function BlueprintsPage() {
  const [selectedBlueprintId, setSelectedBlueprintId] = useState<string | null>(
    null,
  );
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [formError, setFormError] = useState<string | null>(null);

  const queryClient = useQueryClient();
  const {
    data,
    error,
    isLoading,
    isFetching,
  } = useBlueprintsQuery({
    onSuccess: (result) => {
      setSelectedBlueprintId((current) => {
        if (!result.blueprints.length) {
          return null;
        }
        if (
          current &&
          result.blueprints.some((blueprint) => blueprint.id === current)
        ) {
          return current;
        }
        return result.blueprints[0]?.id ?? null;
      });
    },
  });

  const blueprints = useMemo(
    () => (data?.blueprints ?? []).slice().sort((a, b) => a.name.localeCompare(b.name)),
    [data?.blueprints],
  );

  const selectedBlueprint = useMemo(() => {
    if (!blueprints.length) {
      return null;
    }
    if (!selectedBlueprintId) {
      return blueprints[0];
    }
    return (
      blueprints.find((blueprint) => blueprint.id === selectedBlueprintId) ??
      blueprints[0]
    );
  }, [blueprints, selectedBlueprintId]);

  const {
    register,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = useForm<BlueprintFormValues>({
    defaultValues: DEFAULT_FORM_VALUES,
  });

  useEffect(() => {
    if (!isFormOpen) {
      return;
    }
    if (formMode === "edit" && selectedBlueprint) {
      reset({
        name: selectedBlueprint.name,
        os: selectedBlueprint.os,
        version: selectedBlueprint.version,
        data: JSON.stringify(selectedBlueprint.data ?? {}, null, 2),
      });
      return;
    }
    reset(DEFAULT_FORM_VALUES);
  }, [formMode, isFormOpen, reset, selectedBlueprint]);

  const createBlueprint = useMutation({
    mutationFn: async (payload: BlueprintPayload) => {
      const response = await fetch(`${API_BASE_URL}/v1/blueprints`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        throw await extractError(response);
      }
      const body = (await response.json()) as { blueprint: BlueprintRecord };
      return body.blueprint;
    },
    onSuccess: (blueprint) => {
      setIsFormOpen(false);
      queryClient.invalidateQueries({ queryKey: ["blueprints"] });
      setSelectedBlueprintId(blueprint.id);
    },
  });

  const updateBlueprint = useMutation({
    mutationFn: async ({
      id,
      payload,
    }: {
      id: string;
      payload: BlueprintPayload;
    }) => {
      const response = await fetch(`${API_BASE_URL}/v1/blueprints/${id}`, {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        throw await extractError(response);
      }
      const body = (await response.json()) as { blueprint: BlueprintRecord };
      return body.blueprint;
    },
    onSuccess: (blueprint) => {
      setIsFormOpen(false);
      queryClient.invalidateQueries({ queryKey: ["blueprints"] });
      setSelectedBlueprintId(blueprint.id);
    },
  });

  const deleteBlueprint = useMutation({
    mutationFn: async (id: string) => {
      const response = await fetch(`${API_BASE_URL}/v1/blueprints/${id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!response.ok) {
        throw await extractError(response);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["blueprints"] });
    },
  });

  const activeMutation =
    createBlueprint.isPending || updateBlueprint.isPending || deleteBlueprint.isPending;

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null);
    let parsedData: Record<string, unknown> = {};
    const rawData = values.data?.trim();
    if (rawData) {
      try {
        parsedData = JSON.parse(rawData) as Record<string, unknown>;
      } catch (parseError) {
        setFormError("Blueprint data must be valid JSON.");
        console.error("Failed to parse blueprint data", parseError);
        return;
      }
    }

    const payload: BlueprintPayload = {
      name: values.name.trim(),
      os: values.os.trim(),
      version: values.version.trim(),
      data: parsedData,
    };

    if (payload.name === "" || payload.os === "" || payload.version === "") {
      setFormError("Name, OS, and version are required.");
      return;
    }

    try {
      if (formMode === "create") {
        await createBlueprint.mutateAsync(payload);
      } else if (selectedBlueprint) {
        await updateBlueprint.mutateAsync({ id: selectedBlueprint.id, payload });
      }
    } catch (mutationError) {
      const message =
        mutationError instanceof Error
          ? mutationError.message
          : "Unable to save blueprint.";
      setFormError(message);
    }
  });

  const handleDelete = async (blueprint: BlueprintRecord) => {
    if (!window.confirm(`Delete blueprint “${blueprint.name}”?`)) {
      return;
    }
    try {
      await deleteBlueprint.mutateAsync(blueprint.id);
    } catch (mutationError) {
      const message =
        mutationError instanceof Error
          ? mutationError.message
          : "Unable to delete blueprint.";
      setFormError(message);
    }
  };

  const mutationIcon: MutationIcon | null = useMemo(() => {
    if (deleteBlueprint.isPending) {
      return Trash2;
    }
    if (updateBlueprint.isPending) {
      return Edit;
    }
    if (createBlueprint.isPending) {
      return Plus;
    }
    return null;
  }, [createBlueprint.isPending, deleteBlueprint.isPending, updateBlueprint.isPending]);

  const ActiveMutationIcon = mutationIcon;

  return (
    <section className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold">Blueprints</h2>
          <p className="text-sm text-muted-foreground">
            Manage provisioning blueprints sourced from the control plane.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            className="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-sm shadow-sm transition-colors hover:bg-muted"
            onClick={() => {
              setFormMode("create");
              setIsFormOpen(true);
              setFormError(null);
            }}
            disabled={activeMutation}
          >
            <Plus className="h-4 w-4" /> New blueprint
          </button>
          <button
            type="button"
            className="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-sm shadow-sm transition-colors hover:bg-muted"
            onClick={() => queryClient.invalidateQueries({ queryKey: ["blueprints"] })}
            disabled={isFetching}
          >
            <RefreshCcw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
            Refresh
          </button>
        </div>
      </div>

      {isFormOpen && (
        <div className="rounded-lg border bg-card p-4 shadow-sm">
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-semibold">
                  {formMode === "create" ? "Create blueprint" : "Edit blueprint"}
                </h3>
                <p className="text-sm text-muted-foreground">
                  Provide metadata and provisioning payload in JSON format.
                </p>
              </div>
              <button
                type="button"
                className="rounded-md px-2 py-1 text-sm text-muted-foreground transition-colors hover:bg-muted"
                onClick={() => {
                  setIsFormOpen(false);
                  setFormError(null);
                }}
                disabled={isSubmitting || activeMutation}
              >
                Close
              </button>
            </div>

            <div className="grid gap-4 md:grid-cols-3">
              <label className="space-y-2 text-sm font-medium">
                Name
                <input
                  className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                  placeholder="rocky/9/base"
                  {...register("name")}
                  disabled={isSubmitting || activeMutation}
                  required
                />
              </label>
              <label className="space-y-2 text-sm font-medium">
                Operating system
                <input
                  className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                  placeholder="Rocky Linux"
                  {...register("os")}
                  disabled={isSubmitting || activeMutation}
                  required
                />
              </label>
              <label className="space-y-2 text-sm font-medium">
                Version
                <input
                  className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                  placeholder="9.4"
                  {...register("version")}
                  disabled={isSubmitting || activeMutation}
                  required
                />
              </label>
            </div>

            <label className="block space-y-2 text-sm font-medium">
              Blueprint data (JSON)
              <textarea
                className="min-h-[200px] w-full rounded-md border bg-background px-3 py-2 font-mono text-xs"
                {...register("data")}
                disabled={isSubmitting || activeMutation}
              />
            </label>

            {formError && (
              <p className="text-sm text-rose-500" role="alert">
                {formError}
              </p>
            )}

            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                className="rounded-md border px-3 py-2 text-sm transition-colors hover:bg-muted"
                onClick={() => {
                  reset(formMode === "edit" && selectedBlueprint
                    ? {
                        name: selectedBlueprint.name,
                        os: selectedBlueprint.os,
                        version: selectedBlueprint.version,
                        data: JSON.stringify(selectedBlueprint.data ?? {}, null, 2),
                      }
                    : DEFAULT_FORM_VALUES);
                  setFormError(null);
                }}
                disabled={isSubmitting || activeMutation}
              >
                Reset
              </button>
              <button
                type="submit"
                className="inline-flex items-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow-sm transition-opacity disabled:opacity-70"
                disabled={isSubmitting || activeMutation}
              >
                {isSubmitting || activeMutation ? (
                  <>
                    {ActiveMutationIcon ? (
                      <ActiveMutationIcon className="h-4 w-4 animate-spin" />
                    ) : (
                      <LoaderCircle className="h-4 w-4 animate-spin" />
                    )}
                    Saving…
                  </>
                ) : formMode === "create" ? (
                  <>
                    <CheckCircle2 className="h-4 w-4" /> Save blueprint
                  </>
                ) : (
                  <>
                    <CheckCircle2 className="h-4 w-4" /> Update blueprint
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-[1.2fr,2fr]">
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold">Available blueprints</h3>
            {isFetching && <LoaderCircle className="h-4 w-4 animate-spin text-muted-foreground" />}
          </div>

          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, index) => (
                <div
                  // biome-ignore lint/suspicious/noArrayIndexKey: static skeleton list
                  key={index}
                  className="h-16 animate-pulse rounded-lg border bg-muted/40"
                />
              ))}
            </div>
          ) : error ? (
            <div className="rounded-lg border border-rose-500/60 bg-rose-500/10 p-4 text-sm text-rose-600">
              Failed to load blueprints: {error.message}
            </div>
          ) : blueprints.length === 0 ? (
            <div className="rounded-lg border bg-muted/30 p-4 text-sm text-muted-foreground">
              No blueprints found. Create one to get started.
            </div>
          ) : (
            <div className="space-y-2">
              {blueprints.map((blueprint) => {
                const isSelected = selectedBlueprint?.id === blueprint.id;
                return (
                  <button
                    key={blueprint.id}
                    type="button"
                    onClick={() => setSelectedBlueprintId(blueprint.id)}
                    className={`w-full rounded-lg border px-4 py-3 text-left transition-colors ${
                      isSelected
                        ? "border-primary bg-primary/10"
                        : "hover:bg-muted/50"
                    }`}
                  >
                    <div className="flex items-center justify-between gap-4">
                      <div>
                        <p className="text-sm font-semibold">{blueprint.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {blueprint.os} · Version {blueprint.version}
                        </p>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        Updated {formatTimestamp(blueprint.updated_at)}
                      </p>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-5 shadow-sm">
            {selectedBlueprint ? (
              <div className="space-y-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <h3 className="text-xl font-semibold">{selectedBlueprint.name}</h3>
                    <p className="text-sm text-muted-foreground">
                      {selectedBlueprint.os} · Version {selectedBlueprint.version}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      className="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-sm shadow-sm transition-colors hover:bg-muted"
                      onClick={() => {
                        setFormMode("edit");
                        setIsFormOpen(true);
                        setFormError(null);
                      }}
                      disabled={activeMutation}
                    >
                      <Edit className="h-4 w-4" /> Edit
                    </button>
                    <button
                      type="button"
                      className="inline-flex items-center gap-2 rounded-md border border-rose-500/60 px-3 py-2 text-sm text-rose-600 shadow-sm transition-colors hover:bg-rose-500/10"
                      onClick={() => handleDelete(selectedBlueprint)}
                      disabled={activeMutation}
                    >
                      <Trash2 className="h-4 w-4" /> Delete
                    </button>
                  </div>
                </div>

                <dl className="grid gap-3 md:grid-cols-2">
                  <div>
                    <dt className="text-xs uppercase tracking-wide text-muted-foreground">
                      Created
                    </dt>
                    <dd className="text-sm font-medium">
                      {formatTimestamp(selectedBlueprint.created_at)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-xs uppercase tracking-wide text-muted-foreground">
                      Updated
                    </dt>
                    <dd className="text-sm font-medium">
                      {formatTimestamp(selectedBlueprint.updated_at)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-xs uppercase tracking-wide text-muted-foreground">
                      Blueprint ID
                    </dt>
                    <dd className="text-sm font-mono">{selectedBlueprint.id}</dd>
                  </div>
                </dl>

                <div>
                  <h4 className="text-sm font-semibold">Provisioning payload</h4>
                  <pre className="mt-2 max-h-80 overflow-auto rounded-md border bg-muted/30 p-3 text-xs leading-relaxed">
                    {JSON.stringify(selectedBlueprint.data ?? {}, null, 2)}
                  </pre>
                </div>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">
                Select a blueprint to inspect details and manage its metadata.
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
