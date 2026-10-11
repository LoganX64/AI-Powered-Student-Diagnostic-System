import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { AlertTriangle, CheckCircle2, Clock, Loader2, XCircle } from "lucide-react";
import { getResultByToken, type StudentResult } from "@/services/student.service";

/**
 * Public result page reached from the emailed link.
 *
 * The token in the URL is the credential, so this route must stay outside
 * ProtectedRoute — the recipient may open the link hours later without a
 * student session. Nothing here may render a link carrying the same token, and
 * no external resource may be loaded, or the token would travel in the Referer
 * header. Everything here is same-origin for that reason.
 */
export function StudentResultPage() {
  const { token = "" } = useParams();
  const [result, setResult] = useState<StudentResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [invalid, setInvalid] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setInvalid(false);
    getResultByToken(token)
      .then((res) => {
        if (cancelled) return;
        // null means the backend refused: unknown, expired or revoked. It does
        // not say which, and neither does this page.
        if (res) setResult(res);
        else setInvalid(true);
      })
      .catch(() => {
        if (!cancelled) setInvalid(true);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [token]);

  if (loading) {
    return (
      <Shell>
        <div className="flex h-40 items-center justify-center gap-2 text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
          <p className="text-sm">Loading your result…</p>
        </div>
      </Shell>
    );
  }

  if (invalid || !result) {
    return (
      <Shell>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <AlertTriangle className="size-5 text-yellow-500" />
              This link is no longer valid
            </CardTitle>
            <CardDescription>
              Result links expire after 15 days. Ask your coach to share it again,
              or sign in to see your released results.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="outline">
              <Link to="/dashboard">Go to your dashboard</Link>
            </Button>
          </CardContent>
        </Card>
      </Shell>
    );
  }

  return (
    <Shell>
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle className="text-lg">{result.test_title || "Test result"}</CardTitle>
              <CardDescription>
                {result.student_name}
                {result.student_code ? ` · ${result.student_code}` : ""}
              </CardDescription>
            </div>
            {result.released_at && (
              <Badge variant="outline" className="flex items-center gap-1">
                <Clock className="size-3" />
                Released {new Date(result.released_at).toLocaleString()}
              </Badge>
            )}
          </div>
        </CardHeader>

        <CardContent className="flex flex-col gap-6">
          <div className="flex flex-wrap items-baseline gap-3">
            <span className="text-4xl font-bold tabular-nums">{result.percentage}%</span>
            <span className="text-muted-foreground">
              {result.marks_earned} out of {result.max_marks} marks
            </span>
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Stat icon={<CheckCircle2 className="size-4 text-green-600" />} label="Correct" value={result.correct} />
            <Stat icon={<XCircle className="size-4 text-red-500" />} label="Incorrect" value={result.wrong} />
            <Stat icon={<AlertTriangle className="size-4 text-yellow-500" />} label="Skipped" value={result.skipped} />
          </div>
        </CardContent>
      </Card>
    </Shell>
  );
}

function Stat({ icon, label, value }: { icon: React.ReactNode; label: string; value: number }) {
  return (
    <div className="flex items-center justify-between rounded-lg border p-4">
      <span className="flex items-center gap-2 text-sm text-muted-foreground">
        {icon}
        {label}
      </span>
      <span className="text-lg font-semibold tabular-nums">{value}</span>
    </div>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="mx-auto flex min-h-screen w-full max-w-2xl flex-col justify-center gap-4 p-4">
      {children}
    </div>
  );
}