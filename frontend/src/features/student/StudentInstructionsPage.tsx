import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Clock, CalendarClock } from "lucide-react";
import { ExamHeader } from "../../components/student/exam-header";
import { Button } from "../../components/ui/button";
import { useExamTimer } from "../../hooks/useExamTimer";

// Exam duration — 1 hour
const EXAM_DURATION_SECONDS = 60 * 60;
const EXAM_DURATION_HOURS = EXAM_DURATION_SECONDS / 3600;

const INSTRUCTIONS = [
  "Read each question carefully before selecting your answer.",
  "Each question carries fixed marks. There may be negative marking for wrong answers.",
  "Do not refresh the page or navigate away during the test — your progress may be lost.",
  "Use the question navigator on the right side of the quiz to jump between questions.",
  "You can mark a question for review and return to it later.",
  "Once you click Accept and Begin, the timer will start and cannot be paused.",
  "Submit your answers before the timer reaches zero. The test will auto-submit on time expiry.",
  "Ensure a stable internet connection throughout the test.",
];

function useCurrentTime() {
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  return now;
}

function formatCurrentTime(date: Date): string {
  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function formatCurrentDate(date: Date): string {
  return date.toLocaleDateString([], {
    weekday: "short",
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

export function StudentInstructionsPage() {
  const navigate = useNavigate();
  const currentTime = useCurrentTime();

  const studentCode = useMemo(
    () => localStorage.getItem("student_code") || "",
    [],
  );

  // Guard: redirect to login if no token
  useEffect(() => {
    const token = localStorage.getItem("student_token");
    if (!token) {
      navigate("/", { replace: true });
    }
  }, [navigate]);

  // Timer does NOT start until the student clicks Accept
  const timeLeft = useExamTimer(
    EXAM_DURATION_SECONDS,
    "exam_timer",
    () => {
      navigate("/submitted", { replace: true });
    },
    false, // started = false on instructions page
  );

  const handleAccept = () => {
    // Mark exam as started so the quiz page picks up a running timer
    sessionStorage.setItem("exam_started", "true");
    navigate("/quiz");
  };

  return (
    <div className="flex min-h-screen flex-col bg-background px-4 py-6 sm:px-8">
      {/* Header — timer is static (not ticking) on this page */}
      <ExamHeader candidateName={studentCode} timeLeft={timeLeft} />

      {/* Exam meta info bar */}
      <div className="mt-4 flex flex-wrap items-center gap-4 rounded-2xl border border-border bg-card px-6 py-3 shadow-sm text-sm">
        {/* Exam duration */}
        <div className="flex items-center gap-2 text-foreground font-medium">
          <Clock className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          <span>
            Exam duration:{" "}
            <span className="font-semibold">
              {EXAM_DURATION_HOURS === 1
                ? "1 hour"
                : `${EXAM_DURATION_HOURS} hours`}
            </span>
          </span>
        </div>

        <span className="hidden sm:block text-border">|</span>

        {/* Current date & time */}
        <div className="flex items-center gap-2 text-muted-foreground">
          <CalendarClock className="h-4 w-4" aria-hidden="true" />
          <span>
            {formatCurrentDate(currentTime)}{" "}
            <span className="font-semibold tabular-nums text-foreground">
              {formatCurrentTime(currentTime)}
            </span>
          </span>
        </div>
      </div>

      {/* Instructions card */}
      <div className="mt-4 flex flex-1 flex-col rounded-2xl border border-border bg-card shadow-sm">
        <div className="flex-1 p-8">
          <h2 className="mb-1 text-base font-semibold text-foreground">
            Instructions, guidelines to work with
          </h2>
          <p className="mb-6 text-sm text-muted-foreground">
            Please read all instructions carefully before beginning the test.
          </p>

          <ol className="space-y-3">
            {INSTRUCTIONS.map((instruction, index) => (
              <li key={index} className="flex gap-3 text-sm text-foreground">
                <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold text-muted-foreground">
                  {index + 1}
                </span>
                <span className="leading-relaxed">{instruction}</span>
              </li>
            ))}
          </ol>
        </div>

        {/* Footer with Accept button */}
        <div className="flex justify-end border-t border-border px-8 py-5">
          <Button size="lg" onClick={handleAccept} className="min-w-[160px]">
            Accept &amp; Begin
          </Button>
        </div>
      </div>
    </div>
  );
}
