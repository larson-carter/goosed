import { Link } from "react-router-dom";

export function LoginPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Sign in</h1>
        <p className="text-sm text-muted-foreground">
          Access your goose’d UI account.
        </p>
      </div>
      <form className="space-y-4">
        <div className="space-y-2">
          <label className="text-sm font-medium" htmlFor="email">
            Email
          </label>
          <input
            id="email"
            name="email"
            type="email"
            autoComplete="email"
            className="w-full rounded-md border px-3 py-2"
          />
        </div>
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm font-medium">
            <label htmlFor="password">Password</label>
            <Link className="text-primary hover:underline" to="/auth/forgot">
              Forgot password?
            </Link>
          </div>
          <input
            id="password"
            name="password"
            type="password"
            autoComplete="current-password"
            className="w-full rounded-md border px-3 py-2"
          />
        </div>
        <button
          type="submit"
          className="w-full rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground"
        >
          Sign in
        </button>
      </form>
      <p className="text-sm text-muted-foreground">
        Need an account?{" "}
        <Link className="text-primary hover:underline" to="/auth/signup">
          Create one
        </Link>
      </p>
    </div>
  );
}
