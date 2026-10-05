import { useEffect, useState } from "react";
import { Navigate, Outlet, useLocation } from "react-router-dom";
import { probeRole } from "@/lib/token";
import { roleFromPath, ROLE_REDIRECT_MAP } from "@/config/routes";

export function ProtectedRoute() {
  const { pathname } = useLocation();
  const requiredRole = roleFromPath(pathname);
  const [state, setState] = useState<"checking" | "ok" | "denied">("checking");

  useEffect(() => {
    let cancelled = false;
    setState("checking");
    if (!requiredRole) return;
    probeRole(requiredRole).then((ok) => {
      if (!cancelled) setState(ok ? "ok" : "denied");
    });
    return () => {
      cancelled = true;
    };
  }, [requiredRole]);

  if (!requiredRole) return <Navigate to="/" replace />;
  if (state === "checking") return null;
  if (state === "ok") return <Outlet />;
  return <Navigate to={ROLE_REDIRECT_MAP[requiredRole]} replace />;
}
