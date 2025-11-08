export function MachinesPage() {
  return (
    <section className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold">Machines</h2>
        <p className="text-sm text-muted-foreground">
          Manage goose’d infrastructure and view their current state.
        </p>
      </div>
      <div className="rounded-lg border bg-card p-4 shadow-sm">
        <p className="text-sm text-muted-foreground">
          Machine inventory will load here once the API integration is enabled.
        </p>
      </div>
    </section>
  );
}
