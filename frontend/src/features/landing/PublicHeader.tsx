import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { BarChart3Icon, LayoutDashboardIcon } from "lucide-react";
import { dashboardForRole } from "@/config/routes";
import { probeSession, type Session } from "@/lib/token";

const ROLE_LABEL: Record<Session["role"], string> = {
  admin: "Admin",
  coach: "Coach",
  super_admin: "Super Admin",
  student: "Student",
};

export function PublicHeader() {
  const [session, setSession] = useState<Session | null>(null);

  useEffect(() => {
    probeSession().then(setSession);
  }, []);

  const dashboard = session ? dashboardForRole(session.role) : null;

  return (
    <header className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-sm">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <Link to="/" className="flex items-center gap-2">
          <BarChart3Icon className="size-6 text-primary" />
          <span className="text-lg font-bold">EduQuant</span>
        </Link>
        <nav className="hidden md:flex items-center gap-6 text-sm text-muted-foreground">
          <a href="/#features" className="hover:text-foreground transition-colors">Features</a>
          <Link to="/about" className="hover:text-foreground transition-colors">About</Link>
        </nav>
        <div className="flex items-center gap-3">
          {session && dashboard ? (
            <>
              <span className="hidden text-sm text-muted-foreground sm:inline">
                Signed in as {ROLE_LABEL[session.role]}
              </span>
              <Button size="sm" asChild>
                <Link to={dashboard}>
                  <LayoutDashboardIcon className="size-4" />
                  Dashboard
                </Link>
              </Button>
            </>
          ) : (
            <>
              <Button variant="ghost" size="sm" asChild>
                <Link to="/student-login">Student Login</Link>
              </Button>
              <Button size="sm" asChild>
                <Link to="/admin-signup">Register Now</Link>
              </Button>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
