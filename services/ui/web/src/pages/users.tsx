export function UsersPage() {
  return (
    <section className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold">Users & Roles</h2>
        <p className="text-sm text-muted-foreground">
          Invite teammates, assign RBAC roles, and manage access.
        </p>
      </div>
      <div className="rounded-lg border bg-card p-4 shadow-sm">
        <p className="text-sm text-muted-foreground">
          The user directory and invite workflow will appear here.
        </p>
      </div>
    </section>
  );
}
