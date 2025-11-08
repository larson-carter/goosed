export function DashboardPage() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-semibold">Dashboard</h2>
      <p className="text-sm text-muted-foreground">
        Overview widgets will appear here to monitor machines, runs, and health
        at a glance.
      </p>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <div key={index} className="rounded-lg border bg-card p-4 shadow-sm">
            <div className="text-sm font-medium text-muted-foreground">
              Widget {index + 1}
            </div>
            <div className="mt-2 text-2xl font-semibold text-card-foreground">
              --
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
