import { useRouteError, useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";

export function RouteError() {
  const error = useRouteError();
  const navigate = useNavigate();
  const message =
    error instanceof Error ? error.message : "Something went wrong.";

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 p-6">
      <p role="alert" className="text-sm text-destructive">
        {message}
      </p>
      <div className="flex gap-2">
        <Button variant="outline" onClick={() => window.location.reload()}>
          Reload
        </Button>
        <Button variant="outline" onClick={() => navigate(-1)}>
          Go back
        </Button>
      </div>
    </div>
  );
}
