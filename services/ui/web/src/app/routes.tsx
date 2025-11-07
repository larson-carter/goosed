import { createBrowserRouter, RouterProvider } from "react-router-dom";

import { AuthLayout } from "../components/ui/auth-layout";
import { DashboardPage } from "../pages/dashboard";
import { MachinesPage } from "../pages/machines";
import { RunsPage } from "../pages/runs";
import { BlueprintsPage } from "../pages/blueprints";
import { ArtifactsPage } from "../pages/artifacts";
import { UsersPage } from "../pages/users";
import { SettingsPage } from "../pages/settings";
import { AuditPage } from "../pages/audit";
import { LoginPage } from "../pages/auth/login";
import { SignupPage } from "../pages/auth/signup";
import { ForgotPasswordPage } from "../pages/auth/forgot";
import { ResetPasswordPage } from "../pages/auth/reset";
import { VerifyEmailPage } from "../pages/auth/verify";
import { AppShell } from "../components/ui/app-shell";

const router = createBrowserRouter([
  {
    path: "/auth",
    element: <AuthLayout />,
    children: [
      { path: "login", element: <LoginPage /> },
      { path: "signup", element: <SignupPage /> },
      { path: "forgot", element: <ForgotPasswordPage /> },
      { path: "reset", element: <ResetPasswordPage /> },
      { path: "verify", element: <VerifyEmailPage /> },
    ],
  },
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "machines", element: <MachinesPage /> },
      { path: "runs", element: <RunsPage /> },
      { path: "blueprints", element: <BlueprintsPage /> },
      { path: "artifacts", element: <ArtifactsPage /> },
      { path: "users", element: <UsersPage /> },
      { path: "settings", element: <SettingsPage /> },
      { path: "audit", element: <AuditPage /> },
    ],
  },
]);

export function AppRoutes() {
  return <RouterProvider router={router} />;
}
