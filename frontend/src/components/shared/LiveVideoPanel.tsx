import { AlertCircle, RefreshCw } from "lucide-react";
import { useLiveVideo } from "@/hooks/useLiveVideo";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

export function LiveVideoPanel({
  studentId,
  studentName,
}: {
  studentId: number;
  studentName: string;
}) {
  const { canvasRef, connected, error, reconnect } =
    useLiveVideo(studentId);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          Live Preview — {studentName}
          <Badge variant={connected ? "default" : "secondary"}>
            {connected ? "LIVE" : "OFFLINE"}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="relative">
          <canvas
            ref={canvasRef}
            className="w-full max-w-[640px] rounded-lg border bg-black"
          />
          {!connected && !error && (
            <div className="absolute inset-0 flex items-center justify-center text-muted-foreground text-sm">
              Connecting...
            </div>
          )}
          {error && (
            <div className="absolute inset-0 flex flex-col items-center justify-center gap-2">
              <AlertCircle className="h-5 w-5 text-destructive" />
              <span className="text-sm text-destructive">{error}</span>
              <Button
                variant="outline"
                size="sm"
                onClick={reconnect}
                className="gap-1"
              >
                <RefreshCw className="h-3 w-3" />
                Retry
              </Button>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
