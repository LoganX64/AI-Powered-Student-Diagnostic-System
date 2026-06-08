import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { CheckCircle } from "lucide-react";

const REDIRECT_AFTER_SECONDS = 120; // 2 minutes

function clearStudentSession() {
  localStorage.removeItem("student_token");
  localStorage.removeItem("student_code");
  sessionStorage.removeItem("exam_timer");
  sessionStorage.removeItem("quiz_answers");
}

export function StudentSubmittedPage() {
  const navigate = useNavigate();
  const [countdown, setCountdown] = useState(REDIRECT_AFTER_SECONDS);

  useEffect(() => {
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
  }, [navigate, countdown]);

  const minutes = Math.floor(countdown / 60);
  const seconds = countdown % 60;

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-xl rounded-2xl border border-border bg-card p-12 shadow-sm text-center">
        {/* Icon */}
        <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
          <CheckCircle
            className="h-8 w-8 text-green-600 dark:text-green-400"
            aria-hidden="true"
          />
        </div>

        {/* Heading */}
        <h1 className="text-xl font-semibold text-foreground">
          Your test has been submitted
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Thank you for completing the assessment. Your answers have been
          recorded successfully.
        </p>

        {/* Countdown */}
        <div className="mt-8 rounded-xl border border-border bg-muted/40 px-6 py-4">
          <p className="text-xs text-muted-foreground">
            You will be redirected to the login page in
          </p>
          <p className="mt-1 text-3xl font-bold tabular-nums text-foreground">
            {String(minutes).padStart(2, "0")}:
            {String(seconds).padStart(2, "0")}
          </p>
        </div>
      </div>
    </div>
  );
}
