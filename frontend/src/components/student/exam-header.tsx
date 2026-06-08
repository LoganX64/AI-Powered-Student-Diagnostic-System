import { cn } from "../../lib/utils";

type ExamHeaderProps = {
  candidateName: string;
  timeLeft: number; // in seconds
  className?: string;
};

function formatTime(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;

  if (h > 0) {
    return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  }
  return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

export function ExamHeader({ candidateName, timeLeft, className }: ExamHeaderProps) {
  const isLow = timeLeft <= 300; // red when <= 5 minutes

  return (
    <div
      className={cn(
        "flex items-center justify-between rounded-2xl border border-border bg-card px-5 py-3 shadow-sm",
        className,
      )}
    >
      <span className="text-sm text-muted-foreground">
        Candidate name:{" "}
        <span className="font-semibold text-foreground">{candidateName}</span>
      </span>

      <span
        className={cn(
          "min-w-[80px] rounded-xl border px-4 py-1.5 text-center text-sm font-semibold tabular-nums",
          isLow
            ? "border-red-300 bg-red-50 text-red-600 dark:border-red-700 dark:bg-red-950 dark:text-red-400"
            : "border-border bg-muted text-foreground",
        )}
      >
        {formatTime(timeLeft)}
      </span>
    </div>
  );
}
