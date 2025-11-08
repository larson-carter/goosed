export function RunsPage() {
  return (
    <section className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold">Runs</h2>
        <p className="text-sm text-muted-foreground">
          Track execution history and inspect timelines for orchestrated jobs.
        </p>
      </div>
      <div className="rounded-lg border bg-card p-4 shadow-sm">
        <p className="text-sm text-muted-foreground">
          Run timelines will surface here with filters and live updates.
        </p>
      </div>
    </section>
  );
}
