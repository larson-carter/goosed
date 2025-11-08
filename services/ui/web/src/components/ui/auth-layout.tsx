import { Outlet } from "react-router-dom";

export function AuthLayout() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-muted">
      <div className="w-full max-w-md rounded-lg bg-background p-8 shadow-lg">
        <Outlet />
      </div>
    </div>
  );
}
