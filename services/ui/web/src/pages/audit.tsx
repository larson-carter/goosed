export function AuditPage() {
  return (
    <section className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold">Audit Log</h2>
        <p className="text-sm text-muted-foreground">
          Review security-sensitive actions captured by the platform.
        </p>
      </div>
      <div className="rounded-lg border bg-card p-4 shadow-sm">
        <p className="text-sm text-muted-foreground">
          Audit events will stream here with filtering controls.
        </p>
      </div>
    </section>
  );
}
