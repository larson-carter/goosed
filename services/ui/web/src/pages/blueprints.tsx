export function BlueprintsPage() {
  return (
    <section className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold">Blueprints</h2>
        <p className="text-sm text-muted-foreground">
          Browse the catalog of available provisioning blueprints.
        </p>
      </div>
      <div className="rounded-lg border bg-card p-4 shadow-sm">
        <p className="text-sm text-muted-foreground">
          Blueprint details will render here with read-only metadata.
        </p>
      </div>
    </section>
  );
}
