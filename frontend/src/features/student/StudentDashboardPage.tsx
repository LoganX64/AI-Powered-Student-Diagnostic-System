import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { BarChart3, Clock, FileText, LogOut, Mail, Phone } from "lucide-react";
import { Button } from "../../components/ui/button";
import { getStudentAssignments } from "../../services/student.service";
import type { Assignment } from "../../services/student.service";

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function StudentDashboardPage() {
  const navigate = useNavigate();

  const studentCode = useMemo(
    () => localStorage.getItem("student_code") || "",
    [],
  );

  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await getStudentAssignments();
        if (!cancelled) setAssignments(data);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : "Failed to load assignments");
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [refreshKey]);

  const handleStartExam = (assignmentId: number) => {
    localStorage.setItem("assignment_id", String(assignmentId));
    navigate("/instructions");
  };

  const handleLogout = () => {
    localStorage.removeItem("student_token");
    localStorage.removeItem("student_code");
    navigate("/", { replace: true });
  };

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="flex min-h-screen flex-col bg-background px-4 py-6 sm:px-8">
      {/* Header */}
      <div className="flex items-center justify-between rounded-2xl border border-border bg-card px-5 py-3 shadow-sm">
        <div className="flex items-center gap-3">
          <BarChart3 className="h-5 w-5 text-primary" />
          <span className="text-sm font-bold text-foreground">EduQuant</span>
        </div>

        <span className="text-sm text-muted-foreground">
          Student:{" "}
          <span className="font-semibold text-foreground">{studentCode}</span>
        </span>

        <Button variant="outline" size="sm" onClick={handleLogout} className="gap-2">
          <LogOut className="h-4 w-4" />
          Logout
        </Button>
      </div>

      {/* Content */}
      <div className="mt-6 flex flex-1 flex-col rounded-2xl border border-border bg-card shadow-sm">
        <div className="border-b border-border px-7 py-5">
          <h1 className="text-lg font-semibold text-foreground">Your Exams</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Select an exam below to begin.
          </p>
        </div>

        <div className="flex-1 p-7">
          {/* Loading state */}
          {loading && (
            <div className="flex flex-col items-center justify-center py-16">
              <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
              <p className="mt-4 text-sm text-muted-foreground">Loading assignments...</p>
            </div>
          )}

          {/* Error state */}
          {!loading && error && (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <div className="rounded-xl border border-red-200 bg-red-50 px-6 py-4 dark:border-red-800 dark:bg-red-950">
                <p className="text-sm font-medium text-red-700 dark:text-red-400">
                  {error}
                </p>
              </div>
              <Button onClick={() => setRefreshKey((k) => k + 1)} className="mt-4" variant="outline">
                Try Again
              </Button>
            </div>
          )}

          {/* Empty state — no assignments */}
          {!loading && !error && assignments.length === 0 && (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <div className="flex h-16 w-16 items-center justify-center rounded-full bg-muted">
                <FileText className="h-8 w-8 text-muted-foreground" />
              </div>
              <h2 className="mt-4 text-base font-semibold text-foreground">
                No exams assigned yet
              </h2>
              <p className="mt-2 max-w-sm text-sm text-muted-foreground">
                You don&apos;t have any exams assigned to you at the moment.
                Please contact your coach or administrator for assistance.
              </p>
              <div className="mt-6 rounded-xl border border-border bg-muted/40 px-6 py-4 text-left">
                <p className="text-xs font-semibold uppercase text-muted-foreground">
                  Need help?
                </p>
                <div className="mt-2 space-y-1.5">
                  <p className="flex items-center gap-2 text-sm text-foreground">
                    <Mail className="h-4 w-4 text-muted-foreground" />
                    Contact your coach or admin
                  </p>
                  <p className="flex items-center gap-2 text-sm text-foreground">
                    <Phone className="h-4 w-4 text-muted-foreground" />
                    Reach out for assignment support
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Assignment list */}
          {!loading && !error && assignments.length > 0 && (
            <div className="space-y-3">
              {assignments.map((assignment) => {
                const isSubmitted = assignment.status === "submitted";
                const assignedDate = new Date(assignment.assigned_at).toLocaleDateString(
                  "en-US",
                  { weekday: "short", year: "numeric", month: "short", day: "numeric" },
                );

                return (
                  <div
                    key={assignment.id}
                    className="flex items-center justify-between rounded-xl border border-border bg-card px-5 py-4 transition-colors hover:bg-muted/40"
                  >
                    <div className="flex items-center gap-4">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                        <FileText className="h-5 w-5 text-primary" />
                      </div>
                      <div>
                        <p className="text-sm font-semibold text-foreground">
                          {assignment.test_title}
                        </p>
                        <div className="mt-0.5 flex items-center gap-3 text-xs text-muted-foreground">
                          <span className="flex items-center gap-1">
                            <Clock className="h-3 w-3" />
                            {assignedDate}
                          </span>
                          <span
                            className={
                              isSubmitted
                                ? "text-green-600 font-medium"
                                : "text-yellow-600 font-medium"
                            }
                          >
                            {isSubmitted ? "Submitted" : "Assigned"}
                          </span>
                        </div>
                      </div>
                    </div>

                    <Button
                      onClick={() => handleStartExam(assignment.id)}
                      disabled={isSubmitted}
                      variant={isSubmitted ? "outline" : "default"}
                      className="min-w-[120px]"
                    >
                      {isSubmitted ? "Completed" : "Start Exam"}
                    </Button>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
