export function ArtifactsPage() {
  return (
    <section className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold">Artifacts</h2>
        <p className="text-sm text-muted-foreground">
          Explore build outputs and evidence collected from runs.
        </p>
      </div>
      <div className="rounded-lg border bg-card p-4 shadow-sm">
        <p className="text-sm text-muted-foreground">
          Artifact search and download options will be displayed here.
        </p>
      </div>
    </section>
  );
}
