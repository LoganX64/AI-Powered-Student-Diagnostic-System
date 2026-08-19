import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { CheckCircle, RefreshCw } from "lucide-react";
import { Button } from "../../components/ui/button";
import { submitAnswers } from "../../services/student.service";
import type { AnswerPayload } from "../../services/student.service";
import { ROLE_CHANGE_EVENT } from "../../hooks/useRole";

const REDIRECT_AFTER_SECONDS = 120; // 2 minutes

function clearStudentSession() {
  localStorage.removeItem("student_token");
  localStorage.removeItem("student_code");
  localStorage.removeItem("assignment_id");
  localStorage.removeItem("exam_started");
  localStorage.removeItem("exam_started_at");
  localStorage.removeItem("exam_timer");
  localStorage.removeItem("exam_duration");
  localStorage.removeItem("quiz_answers");
  localStorage.removeItem("quiz_answer_details");
  localStorage.removeItem("current_question_index");
  localStorage.removeItem("pending_submission");
  for (let i = localStorage.length - 1; i >= 0; i--) {
    const k = localStorage.key(i);
    if (k && k.startsWith("exam_ctx_")) localStorage.removeItem(k);
  }
  window.dispatchEvent(new Event(ROLE_CHANGE_EVENT));
}

type PendingSubmission = {
  assignment_id: number;
  answers: AnswerPayload[];
  queued_at: number;
};

function loadPendingSubmission(): PendingSubmission | null {
  try {
    const raw = localStorage.getItem("pending_submission");
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function StudentSubmittedPage() {
  const navigate = useNavigate();
  const [countdown, setCountdown] = useState(REDIRECT_AFTER_SECONDS);
  const [retrying, setRetrying] = useState(false);
  const [retrySuccess, setRetrySuccess] = useState(false);
  const [pending, setPending] = useState<PendingSubmission | null>(
    loadPendingSubmission,
  );

  const hasPendingSubmission = pending !== null && !retrySuccess;

  const retryPendingSubmission = async () => {
    if (!pending || retrying) return;
    setRetrying(true);

    try {
      await submitAnswers(pending.assignment_id, pending.answers);
      localStorage.removeItem("pending_submission");
      setPending(null);
      setRetrySuccess(true);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      if (msg.includes("already submitted")) {
        localStorage.removeItem("pending_submission");
        setPending(null);
        setRetrySuccess(true);
        return;
      }
    } finally {
      setRetrying(false);
    }
  };

  // Auto-retry pending submission on mount.
  useEffect(() => {
    if (pending && !retrying && !retrySuccess) {
      const id = window.setTimeout(() => {
        retryPendingSubmission();
      }, 0);

      return () => window.clearTimeout(id);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (hasPendingSubmission) {
      return;
    }

    if (countdown <= 0) {
      clearStudentSession();
      navigate("/", { replace: true });
      return;
    }

    const id = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(id);
          clearStudentSession();
          navigate("/", { replace: true });
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(id);
  }, [navigate, countdown, hasPendingSubmission]);

  const handleRedirectNow = () => {
    clearStudentSession();
    navigate("/", { replace: true });
  };

  const minutes = Math.floor(countdown / 60);
  const seconds = countdown % 60;

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-xl rounded-2xl border border-border bg-card p-12 shadow-sm text-center">
        <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-green-100 ">
          <CheckCircle
            className="h-8 w-8 text-green-600 "
            aria-hidden="true"
          />
        </div>

        <h1 className="text-xl font-semibold text-foreground">
          {hasPendingSubmission ? "Submission pending" : "Your test has been submitted"}
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          {hasPendingSubmission
            ? "Your answers are saved on this device. Keep this page open while we retry."
            : "Thank you for completing the assessment. Your answers have been recorded successfully."}
        </p>

        {hasPendingSubmission && (
          <div className="mt-6 rounded-xl border border-yellow-300 bg-yellow-50 px-6 py-4  ">
            <p className="text-sm text-yellow-800 ">
              Your submission is pending. Click below to retry.
            </p>
            <Button
              onClick={retryPendingSubmission}
              disabled={retrying}
              className="mt-3 gap-2"
              variant="outline"
            >
              <RefreshCw
                className={`h-4 w-4 ${retrying ? "animate-spin" : ""}`}
              />
              {retrying ? "Retrying..." : "Retry Submission"}
            </Button>
          </div>
        )}

        {retrySuccess && (
          <div className="mt-6 rounded-xl border border-green-300 bg-green-50 px-6 py-4  ">
            <p className="text-sm text-green-800 ">
              Submission successful!
            </p>
          </div>
        )}

        {!hasPendingSubmission && (
          <>
            <div className="mt-8 rounded-xl border border-border bg-muted/40 px-6 py-4">
              <p className="text-xs text-muted-foreground">
                You will be redirected to the home page in
              </p>
              <p className="mt-1 text-3xl font-bold tabular-nums text-foreground">
                {String(minutes).padStart(2, "0")}:
                {String(seconds).padStart(2, "0")}
              </p>
            </div>

            <Button
              variant="outline"
              className="mt-6 min-w-[160px]"
              onClick={handleRedirectNow}
            >
              Redirect Now
            </Button>
          </>
        )}
      </div>
    </div>
  );
}
