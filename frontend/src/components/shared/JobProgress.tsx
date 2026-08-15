import { useEffect, useRef, useState } from "react";
import { Loader2, CheckCircle2, XCircle } from "lucide-react";
import { getJob, type Job } from "@/services/dashboard.service";

export function JobProgress({
  jobId,
  label,
  onDone,
}: {
  jobId: number;
  label?: string;
  onDone?: (job: Job) => void;
}) {
  const [job, setJob] = useState<Job | null>(null);
  const [error, setError] = useState<string | null>(null);
  const onDoneRef = useRef(onDone);
  onDoneRef.current = onDone;

  useEffect(() => {
    let active = true;
    let timer: ReturnType<typeof setInterval> | null = null;

    const poll = async () => {
      try {
        const j = await getJob(jobId);
        if (!active) return;
        setJob(j);
        if (j.status === "completed" || j.status === "failed") {
          if (timer) clearInterval(timer);
          onDoneRef.current?.(j);
        }
      } catch (e) {
        if (!active) return;
        setError((e as Error).message);
        if (timer) clearInterval(timer);
      }
    };

    poll();
    timer = setInterval(poll, 1500);
    return () => {
      active = false;
      if (timer) clearInterval(timer);
    };
  }, [jobId]);

  const pct =
    job && job.total > 0 ? Math.round((job.done / job.total) * 100) : 0;

  return (
    <div className="rounded-lg border p-3 flex flex-col gap-2">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">
          {label ? `${label}…` : "Computing…"}{" "}
          {job ? `${job.done}/${job.total}` : ""}
        </span>
        {job?.status === "completed" && (
          <CheckCircle2 className="size-4 text-green-600" />
        )}
        {job?.status === "failed" && <XCircle className="size-4 text-destructive" />}
        {!job && <Loader2 className="size-4 animate-spin" />}
      </div>
      <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
        <div
          className="h-full bg-primary transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      {job?.failed > 0 && (
        <p className="text-xs text-destructive">{job.failed} failed</p>
      )}
      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  );
}
