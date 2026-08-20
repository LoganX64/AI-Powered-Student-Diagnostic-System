import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeftIcon, BarChart3Icon, Loader2Icon } from "lucide-react";
import { toast } from "sonner";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { AnalysisDashboard } from "@/components/shared/AnalysisDashboard";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getStudentSQI, computeSQI } from "@/services/dashboard.service";
import { parseRouteId } from "@/lib/utils";
import type { SQIResponse } from "@/services/types";
import { JobProgress } from "@/components/shared/JobProgress";

function sqiColor(score: number): string {
  if (score >= 75) return "text-green-600";
  if (score >= 50) return "text-yellow-600";
  return "text-red-600";
}

export function StudentSQIPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const role = useRole();
  const studentId = parseRouteId(id);
  const prefix = role === "admin" ? "/admin" : "/coach";

  const [data, setData] = useState<SQIResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [computeJobId, setComputeJobId] = useState<number | null>(null);

  const handleCompute = async (attemptId: number) => {
    try {
      const { job_id } = await computeSQI(attemptId);
      setComputeJobId(job_id);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const reload = useCallback(async () => {
    if (studentId === null) return;
    try {
      const res = await getStudentSQI(studentId, { include_analysis: true });
      setData(res);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load SQI data");
    }
  }, [studentId]);

  useEffect(() => {
    if (studentId === null) return;
    let cancelled = false;
    const sid = studentId;
    async function load() {
      try {
        const res = await getStudentSQI(sid, { include_analysis: true });
        if (!cancelled) setData(res);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : "Failed to load SQI data");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, [studentId, reload]);

  if (studentId === null) {
    return (
      <DashboardLayout title="Student Not Found">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Invalid student ID in URL.</p>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout title="SQI Score">
      <Button
        variant="ghost"
        size="sm"
        className="w-fit"
        onClick={() => navigate(`${prefix}/students/${studentId}`)}
      >
        <ArrowLeftIcon className="size-4 mr-2" /> Back to Student Detail
      </Button>

      <div className="flex items-center gap-3">
        <BarChart3Icon className="size-6 text-muted-foreground" />
        <h1 className="text-2xl font-bold">SQI Score</h1>
        {data?.name && <Badge variant="secondary">{data.name}</Badge>}
      </div>

      {loading && (
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2Icon className="size-4 animate-spin" /> Loading SQI data...
        </div>
      )}

      {error && (
        <Card>
          <CardContent className="py-6 text-center text-destructive">{error}</CardContent>
        </Card>
      )}

      {!loading && !error && data && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>Summary</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex gap-8">
                <div>
                  <p className="text-sm text-muted-foreground">Average SQI</p>
                  <p className={`text-3xl font-bold ${data.average_sqi != null ? sqiColor(data.average_sqi) : ""}`}>
                    {data.average_sqi != null ? data.average_sqi.toFixed(1) : "—"}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Total Tests</p>
                  <p className="text-3xl font-bold">{data.total_tests}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <div className="flex flex-col gap-3">
            <h2 className="text-base font-semibold">Attempts</h2>
            {computeJobId !== null && (
              <JobProgress
                jobId={computeJobId}
                label="Calculating SQI"
                onDone={() => {
                  setComputeJobId(null);
                  reload();
                }}
              />
            )}
            {(data.attempts ?? []).length === 0 && (
              <p className="text-sm text-muted-foreground">No attempts found.</p>
            )}
            {(data.attempts ?? []).map((attempt) => (
              <Card key={attempt.attempt_id}>
                <CardHeader>
                  <CardTitle className="flex items-center justify-between gap-2">
                    <span className="truncate">Attempt #{attempt.attempt_id}</span>
                    <span className="flex items-center gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={computeJobId !== null}
                        onClick={() => handleCompute(attempt.attempt_id)}
                      >
                        Calculate Score
                      </Button>
                      <Badge variant="outline" className={`font-mono ${sqiColor(attempt.sqi_score)}`}>
                        SQI: {attempt.sqi_score.toFixed(1)}
                      </Badge>
                    </span>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex gap-6 text-sm">
                    <div>
                      <span className="text-muted-foreground">Attempt ID: </span>
                      <span className="font-mono">{attempt.attempt_id}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Test ID: </span>
                      <span className="font-mono">{attempt.test_id}</span>
                    </div>
                  </div>
                  {attempt.analysis && (
                    <details className="mt-3">
                      <summary className="cursor-pointer text-sm text-muted-foreground hover:text-foreground">
                        View Analysis
                      </summary>
                      <div className="mt-2">
                        <AnalysisDashboard data={attempt.analysis} />
                      </div>
                    </details>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        </>
      )}

      {!loading && !error && data && (data.attempts ?? []).length === 0 && (
        <Card>
          <CardContent className="py-6 text-center text-muted-foreground">
            No SQI data available. The student hasn&apos;t submitted any tests yet.
          </CardContent>
        </Card>
      )}
    </DashboardLayout>
  );
}
