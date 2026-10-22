import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useSession } from "@/contexts/SessionContext";
import {
  dashboardForRole,
  roleFromPath,
  ROLE_REDIRECT_MAP,
} from "@/config/routes";
import { Button } from "@/components/ui/button";

export function ProtectedRoute() {
  const { pathname } = useLocation();
  const requiredRole = roleFromPath(pathname);
  const { status, session, refresh } = useSession();

  if (!requiredRole) return <Navigate to="/" replace />;

  if (status === "loading") return null;

  if (status === "error") {
    return (
      <div className="flex min-h-screen items-center justify-center px-4">
        <div className="max-w-sm text-center">
          <h1 className="text-lg font-semibold">Could not verify your session</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            The server is unreachable. Check your connection and try again.
          </p>
          <Button className="mt-4" onClick={() => void refresh()}>
            Try again
          </Button>
        </div>
      </div>
    );
  }

  if (!session) {
    return <Navigate to={ROLE_REDIRECT_MAP[requiredRole]} replace />;
  }

  if (!session.roles.includes(requiredRole)) {
    return <Navigate to={dashboardForRole(session.role)} replace />;
  }

  return <Outlet />;
}
