import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeftIcon, BarChart3Icon, CheckCircleIcon, XCircleIcon, ClockIcon, EyeIcon, SendIcon, CheckIcon, CopyIcon } from "lucide-react";
import { toast } from "sonner";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { JobProgress } from "@/components/shared/JobProgress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  getAssignmentDetail,
  computeSQI,
  releaseResults,
  type AssignmentDetail,
} from "@/services/dashboard.service";
import { formatDateDDMMYYYY, parseRouteId } from "@/lib/utils";
import { RecordedVideoPlayer } from "@/components/shared/RecordedVideoPlayer";

export function AssignmentDetailPage() {
  const { id, assignmentId } = useParams<{ id: string; assignmentId: string }>();
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";
  const studentId = parseRouteId(id);
  const parsedAssignmentId = parseRouteId(assignmentId);

  const [data, setData] = useState<AssignmentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [computeJobId, setComputeJobId] = useState<number | null>(null);
  const [videoDeleted, setVideoDeleted] = useState(false);
  const [releaseOpen, setReleaseOpen] = useState(false);
  const [releasing, setReleasing] = useState(false);
  // Set when the release succeeded but no email went out, so the coach can
  // copy the link manually.
  const [shareLink, setShareLink] = useState<string | null>(null);

  const reload = useCallback(() => {
    if (studentId === null || parsedAssignmentId === null) return;
    getAssignmentDetail(studentId, parsedAssignmentId)
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load"))
      .finally(() => setLoading(false));
  }, [studentId, parsedAssignmentId]);

  const handleCompute = async () => {
    const attemptId = data?.attempt?.id;
    if (attemptId == null) return;
    try {
      const { job_id } = await computeSQI(attemptId);
      setComputeJobId(job_id);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  useEffect(() => {
    if (studentId === null || parsedAssignmentId === null) return;
    setLoading(true);
    getAssignmentDetail(studentId, parsedAssignmentId)
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load"))
      .finally(() => setLoading(false));
  }, [studentId, parsedAssignmentId]);

  const handleRelease = async () => {
    if (studentId === null || parsedAssignmentId === null) return;
    try {
      setReleasing(true);
      const res = await releaseResults(studentId, parsedAssignmentId);
      if (res.already_released) {
        toast.info("This result was already released.");
      } else if (res.email_sent) {
        toast.success(`Result released. Email sent to ${data?.student.name}.`);
      } else {
        toast.success("Result released. No email was sent - share the link below.");
        setShareLink(res.link ?? null);
      }
      setReleaseOpen(false);
      reload();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setReleasing(false);
    }
  };

  if (studentId === null || parsedAssignmentId === null) {
    return (
      <DashboardLayout title="Assignment Not Found">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Invalid assignment ID in URL.</p>
        </div>
      </DashboardLayout>
    );
  }

  if (loading) {
    return (
      <DashboardLayout title="Assignment Detail">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </DashboardLayout>
    );
  }

  if (error || !data) {
    return (
      <DashboardLayout title="Assignment Detail">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-red-500">{error || "Not found"}</p>
        </div>
      </DashboardLayout>
    );
  }

  const { student, test, assignment, attempt, sqi_score, answers } = data;

  const totalQuestions = answers.length;
  const correctCount = answers.filter((a) => a.is_correct).length;
  const wrongCount = answers.filter((a) => a.seen && a.selected_answer !== "" && !a.is_correct).length;
  const unansweredCount = answers.filter((a) => !a.seen || a.selected_answer === "").length;
  const totalTime = answers.reduce((sum, a) => sum + a.time_spent, 0);

  return (
    <DashboardLayout title={`Assignment — ${student.name}`}>
      {/* Back button */}
      <Button
        variant="ghost"
        size="sm"
        className="w-fit"
        onClick={() => navigate(`${prefix}/students/${studentId}`)}
      >
        <ArrowLeftIcon className="size-4 mr-2" /> Back to Student
      </Button>

      {/* Student + Test info */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-3 flex-wrap">
            {student.name}
            <Badge variant="outline" className="font-mono">{student.student_code}</Badge>
            <Badge variant={assignment.status === "submitted" ? "default" : "secondary"}>
              {assignment.status}
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-4 text-sm">
            <div>
              <span className="text-muted-foreground">Test: </span>
              <span className="font-medium">{test.title}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Assigned: </span>
              <span className="font-medium">{formatDateDDMMYYYY(assignment.assigned_at)}</span>
            </div>
            {attempt && attempt.submitted_at && (
              <div>
                <span className="text-muted-foreground">Submitted: </span>
                <span className="font-medium">{formatDateDDMMYYYY(attempt.submitted_at)}</span>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Summary cards */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Card>
          <CardContent className="p-4 text-center">
            <p className="text-2xl font-bold">{totalQuestions}</p>
            <p className="text-xs text-muted-foreground">Total Questions</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <p className="text-2xl font-bold text-green-600">{correctCount}</p>
            <p className="text-xs text-muted-foreground">Correct</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <p className="text-2xl font-bold text-red-600">{wrongCount}</p>
            <p className="text-xs text-muted-foreground">Wrong</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <p className="text-2xl font-bold text-yellow-600">{unansweredCount}</p>
            <p className="text-xs text-muted-foreground">Unanswered</p>
          </CardContent>
        </Card>
      </div>

      {/* SQI Score + Time */}
      <div className="flex items-center gap-3 flex-wrap">
        {sqi_score != null && sqi_score > 0 && (
          <Button
            variant="outline"
            className="w-fit gap-2"
            onClick={() => navigate(`${prefix}/students/${studentId}/sqi`)}
          >
            <BarChart3Icon className="size-4" /> SQI Score: {sqi_score}
          </Button>
        )}
        <Button
          variant="outline"
          className="w-fit gap-2"
          disabled={!data?.attempt || computeJobId !== null}
          onClick={handleCompute}
        >
          <BarChart3Icon className="size-4" /> Calculate Score
        </Button>
        <ReleaseButton
          computed={data.result_computed === true}
          releasedAt={data.released_at ?? null}
          computePending={computeJobId !== null}
          onOpen={() => setReleaseOpen(true)}
        />
        <span className="text-sm text-muted-foreground">
          <ClockIcon className="size-3.5 inline mr-1" />
          Total time: {Math.floor(totalTime / 60)}m {Math.round(totalTime % 60)}s
        </span>
      </div>

      <Dialog open={releaseOpen} onOpenChange={setReleaseOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Release result to {student.name}?</DialogTitle>
            <DialogDescription>
              {student.name} will be able to see their marks and percentage
              for this attempt. The link expires after 15 days. If they have an
              email on file it will be sent to them; otherwise you will get a
              link to share. This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setReleaseOpen(false)} disabled={releasing}>
              Cancel
            </Button>
            <Button onClick={handleRelease} disabled={releasing}>
              {releasing ? "Releasing..." : "Release result"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={shareLink !== null}
        onOpenChange={(o) => !o && setShareLink(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Share this result link</DialogTitle>
            <DialogDescription>
              No email was sent, so send this link to {student.name} yourself.
              It expires in 15 days.
            </DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2">
            <code className="flex-1 overflow-x-auto rounded border bg-muted px-2 py-1 text-xs">
              {shareLink}
            </code>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                navigator.clipboard
                  ?.writeText(shareLink ?? "")
                  .then(() => toast.success("Link copied"))
                  .catch(() => toast.error("Could not copy"));
              }}
            >
              <CopyIcon className="size-4" /> Copy
            </Button>
          </div>
        </DialogContent>
      </Dialog>

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

      {/* Recorded Video Playback */}
      {assignment.status === "submitted" && !videoDeleted && (
        <RecordedVideoPlayer
          assignmentId={parsedAssignmentId}
          title={`Recorded Video — ${student.name}`}
          onDelete={() => setVideoDeleted(true)}
        />
      )}

      {/* Answers table */}
      {answers.length === 0 ? (
        <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
          <p className="text-sm text-muted-foreground">No answers recorded (student did not attempt).</p>
        </div>
      ) : (
        <div className="rounded-lg border overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">#</TableHead>
                <TableHead>Question</TableHead>
                <TableHead className="w-20">Student</TableHead>
                <TableHead className="w-20">Correct</TableHead>
                <TableHead className="w-16">Result</TableHead>
                <TableHead className="w-20">Time</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {answers.map((a, idx) => (
                <TableRow key={a.question_id}>
                  <TableCell className="font-mono text-sm text-muted-foreground">{idx + 1}</TableCell>
                  <TableCell>
                    <p className="text-sm font-medium line-clamp-2">{a.question_text}</p>
                    <div className="mt-1 flex flex-wrap gap-1 text-xs text-muted-foreground">
                      <span className={a.selected_answer === "A" ? "font-bold text-foreground" : ""}>A: {a.option_a}</span>
                      <span> | </span>
                      <span className={a.selected_answer === "B" ? "font-bold text-foreground" : ""}>B: {a.option_b}</span>
                      <span> | </span>
                      <span className={a.selected_answer === "C" ? "font-bold text-foreground" : ""}>C: {a.option_c}</span>
                      <span> | </span>
                      <span className={a.selected_answer === "D" ? "font-bold text-foreground" : ""}>D: {a.option_d}</span>
                    </div>
                    {!a.seen && <Badge variant="secondary" className="mt-1 text-xs">Not Seen</Badge>}
                    {a.marked_for_review && <Badge variant="outline" className="mt-1 text-xs ml-1">Review</Badge>}
                  </TableCell>
                  <TableCell className="text-center">
                    <Badge variant={a.selected_answer ? "outline" : "secondary"}>
                      {a.selected_answer || "—"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-center">
                    <Badge variant="outline" className="font-bold">{a.correct_answer}</Badge>
                  </TableCell>
                  <TableCell className="text-center">
                    {!a.seen ? (
                      <EyeIcon className="size-4 mx-auto text-muted-foreground" />
                    ) : a.is_correct ? (
                      <CheckCircleIcon className="size-4 mx-auto text-green-600" />
                    ) : a.selected_answer === "" ? (
                      <span className="text-xs text-muted-foreground">—</span>
                    ) : (
                      <XCircleIcon className="size-4 mx-auto text-red-600" />
                    )}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground tabular-nums">
                    {a.time_spent > 0 ? `${a.time_spent.toFixed(1)}s` : "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </DashboardLayout>
  );
}

/**
 * Release is gated on `result_computed` rather than on `sqi_score` being
 * non-null: a student who scored zero has a perfectly releasable result, and
 * gating on the score would permanently lock them out of seeing it.
 *
 * The disabled state carries the reason as a tooltip, because a coach hitting a
 * dead button with no explanation would just click "Calculate Score" and hope.
 */
function ReleaseButton({
  computed,
  releasedAt,
  computePending,
  onOpen,
}: {
  computed: boolean;
  releasedAt: string | null;
  computePending: boolean;
  onOpen: () => void;
}) {
  if (releasedAt) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="outline" className="w-fit gap-2" disabled>
              <CheckIcon className="size-4" /> Result released
            </Button>
          </TooltipTrigger>
          <TooltipContent>Released on {formatDateDDMMYYYY(releasedAt)}</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  const disabledReason = computePending
    ? "Waiting for the score to finish calculating."
    : "Calculate the score first — results can only be released once computed.";

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex">
            <Button
              className="w-fit gap-2"
              disabled={!computed || computePending}
              onClick={onOpen}
            >
              <SendIcon className="size-4" /> Release results
            </Button>
          </span>
        </TooltipTrigger>
        {!computed && (
          <TooltipContent>{disabledReason}</TooltipContent>
        )}
      </Tooltip>
    </TooltipProvider>
  );
}
