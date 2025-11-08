import { NavLink, Outlet } from "react-router-dom";
import { Menu, Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { useState } from "react";

const navItems = [
  { to: "/", label: "Dashboard" },
  { to: "/machines", label: "Machines" },
  { to: "/runs", label: "Runs" },
  { to: "/blueprints", label: "Blueprints" },
  { to: "/artifacts", label: "Artifacts" },
  { to: "/users", label: "Users" },
  { to: "/settings", label: "Settings" },
  { to: "/audit", label: "Audit" },
];

export function AppShell() {
  const { theme, setTheme } = useTheme();
  const currentTheme = theme ?? "light";
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const toggleTheme = () => {
    setTheme(currentTheme === "dark" ? "light" : "dark");
  };

  return (
    <div className="flex min-h-screen bg-background text-foreground">
      <button
        type="button"
        className="md:hidden fixed top-4 left-4 z-50 rounded-md border bg-background p-2 shadow"
        onClick={() => setSidebarOpen((prev) => !prev)}
        aria-label="Toggle navigation"
      >
        <Menu className="h-5 w-5" />
      </button>
      <aside
        className={`${
          sidebarOpen ? "translate-x-0" : "-translate-x-full"
        } md:translate-x-0 fixed md:static inset-y-0 left-0 z-40 w-64 bg-muted/40 backdrop-blur transition-transform duration-200`}
      >
        <div className="flex h-full flex-col gap-4 p-6">
          <div className="text-2xl font-semibold">goose’d UI</div>
          <nav className="flex flex-col gap-2">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === "/"}
                className={({ isActive }) =>
                  `rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                    isActive
                      ? "bg-primary text-primary-foreground"
                      : "hover:bg-muted"
                  }`
                }
                onClick={() => setSidebarOpen(false)}
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
        </div>
      </aside>
      <main className="flex-1 md:ml-64">
        <header className="flex items-center justify-between border-b bg-background/70 px-6 py-4 backdrop-blur">
          <div>
            <p className="text-sm text-muted-foreground">Welcome back</p>
            <h1 className="text-xl font-semibold">goose’d Platform</h1>
          </div>
          <button
            type="button"
            className="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-sm shadow-sm"
            onClick={toggleTheme}
          >
            {currentTheme === "dark" ? (
              <>
                <Sun className="h-4 w-4" /> Light mode
              </>
            ) : (
              <>
                <Moon className="h-4 w-4" /> Dark mode
              </>
            )}
          </button>
        </header>
        <div className="px-6 py-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
