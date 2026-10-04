import { Link, useLocation } from "react-router-dom";
import { HomeIcon, CompassIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

function homeFor(pathname: string): string {
  if (pathname.startsWith("/super-admin")) return "/super-admin/dashboard";
  if (pathname.startsWith("/admin")) return "/admin/dashboard";
  if (pathname.startsWith("/coach")) return "/coach/dashboard";
  return "/";
}

export function NotFoundPage() {
  const { pathname } = useLocation();

  const dashboard = homeFor(pathname);
  const isHome = dashboard === "/";

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 px-4 py-10">
      <Card className="w-full max-w-lg shadow-sm">
        {/* font-sans is explicit because CardTitle inherits the app's
            monospace --font-heading, which reads wrong for body copy. */}
        <CardContent className="flex flex-col items-center gap-7 p-8 text-center sm:p-10">
          <div className="flex size-14 items-center justify-center rounded-full bg-muted">
            <CompassIcon className="size-7 text-muted-foreground" aria-hidden="true" />
          </div>

          <div className="flex flex-col items-center gap-3">
            <p className="font-heading text-6xl leading-none font-semibold tracking-tight text-muted-foreground/40">
              404
            </p>
            <h1 className="font-sans text-2xl font-semibold tracking-tight">
              Page not found
            </h1>
            <p className="max-w-sm text-sm text-muted-foreground">
              {isHome
                ? "We couldn't find that page. It may have been moved or removed."
                : "We couldn't find that page. Check the address, or head back to your dashboard."}
            </p>
            <code className="mt-1 block max-w-full rounded-md bg-muted px-3 py-1.5 font-mono text-xs break-all text-muted-foreground">
              {pathname}
            </code>
          </div>

          <div className="flex w-full flex-col items-center gap-2">
            <Button asChild className="w-full gap-2 sm:w-auto">
              <Link to={dashboard}>
                <HomeIcon className="size-4" />
                {isHome ? "Go to home" : "Go to dashboard"}
              </Link>
            </Button>

            {/* Only offered when it leads somewhere different — otherwise the two
                buttons are the same destination with different labels. */}
            {!isHome && (
              <Button asChild variant="ghost" className="text-muted-foreground">
                <Link to="/">Back to home</Link>
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}