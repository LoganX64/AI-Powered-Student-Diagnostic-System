import { Link, useLocation } from "react-router-dom";
import { HomeIcon, CompassIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

function homeFor(pathname: string): string {
  if (pathname.startsWith("/super-admin")) return "/super-admin/dashboard";
  if (pathname.startsWith("/admin")) return "/admin/dashboard";
  if (pathname.startsWith("/coach")) return "/coach/dashboard";
  return "/";
}

export function NotFoundPage() {
  const { pathname } = useLocation();

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-xl">
            <CompassIcon className="size-5" />
            Page not found
          </CardTitle>
          <CardDescription>
            Nothing lives at <span className="font-mono">{pathname}</span>.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Button asChild className="w-fit gap-2">
            <Link to={homeFor(pathname)}>
              <HomeIcon className="size-4" />
              Go to dashboard
            </Link>
          </Button>
          <Button asChild variant="outline" className="w-fit">
            <Link to="/">Back to home</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}