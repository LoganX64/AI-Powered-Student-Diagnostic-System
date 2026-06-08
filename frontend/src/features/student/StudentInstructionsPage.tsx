import { useEffect, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { ExamHeader } from "../../components/student/exam-header";
import { Button } from "../../components/ui/button";
import { useExamTimer } from "../../hooks/useExamTimer";

// Default exam duration: 60 minutes
const EXAM_DURATION_SECONDS = 60 * 60;

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

export function StudentInstructionsPage() {
  const navigate = useNavigate();

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

  // Timer shown on instruction page (counts down even before quiz starts)
  const timeLeft = useExamTimer(EXAM_DURATION_SECONDS, "exam_timer", () => {
    // If time runs out on the instruction page, go straight to submitted
    navigate("/submitted", { replace: true });
  });

  const handleAccept = () => {
    navigate("/quiz");
  };

  return (
    <div className="flex min-h-screen flex-col bg-background px-4 py-6 sm:px-8">
      {/* Header */}
      <ExamHeader candidateName={studentCode} timeLeft={timeLeft} />

      {/* Instructions card */}
      <div className="mt-6 flex flex-1 flex-col rounded-2xl border border-border bg-card shadow-sm">
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
          <Button size="lg" onClick={handleAccept} className="min-w-[140px]">
            Accept
          </Button>
        </div>
      </div>
    </div>
  );
}
