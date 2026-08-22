import { useEffect, useState, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Clock, CalendarClock, AlertTriangle, ArrowLeft, Video, VideoOff } from "lucide-react";
import { ExamHeader } from "../../components/student/exam-header";
import { Button } from "../../components/ui/button";
import { useExamTimer } from "../../hooks/useExamTimer";
import { getAssignmentQuestions } from "../../services/student.service";
import type { AssignmentQuestionsResponse } from "../../services/student.service";

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

type CameraStatus = "idle" | "checking" | "connected" | "failed";

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function StudentInstructionsPage() {
  const navigate = useNavigate();
  const [now, setNow] = useState(() => new Date());

  const [data, setData] = useState<AssignmentQuestionsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [cameraStatus, setCameraStatus] = useState<CameraStatus>("idle");
  const [cameraError, setCameraError] = useState<string | null>(null);
  const cameraStreamRef = useRef<MediaStream | null>(null);
  const videoPreviewRef = useRef<HTMLVideoElement>(null);

  const assignmentId = localStorage.getItem("assignment_id");
  const videoProctoring = data?.integrity_policy?.video_proctoring ?? false;

  // Fetch assignment data
  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!assignmentId) {
        if (!cancelled) {
          setError("No assignment selected. Please go back to the dashboard.");
          setLoading(false);
        }
        return;
      }

      setLoading(true);
      setError(null);
      try {
        const result = await getAssignmentQuestions(Number(assignmentId));
        if (!cancelled) {
          setData(result);
          localStorage.setItem("exam_duration", String(result.duration));
        }
      } catch (err) {
        if (!cancelled) {
          const msg = err instanceof Error ? err.message : "Failed to load exam";
          setError(msg);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [assignmentId]);

  // Camera check: when data loads and video_proctoring is enabled, test the camera.
  const checkCamera = useCallback(async () => {
    if (!videoProctoring) return;
    setCameraStatus("checking");
    setCameraError(null);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
      cameraStreamRef.current = stream;
      if (videoPreviewRef.current) {
        videoPreviewRef.current.srcObject = stream;
      }
      setCameraStatus("connected");
    } catch (err) {
      setCameraStatus("failed");
      const msg = err instanceof Error ? err.message : "Camera access denied";
      setCameraError(msg);
    }
  }, [videoProctoring]);

  useEffect(() => {
    if (data && videoProctoring && cameraStatus === "idle") {
      checkCamera();
    }
  }, [data, videoProctoring, cameraStatus, checkCamera]);

  // Cleanup camera on unmount
  useEffect(() => {
    return () => {
      if (cameraStreamRef.current) {
        cameraStreamRef.current.getTracks().forEach((t) => t.stop());
        cameraStreamRef.current = null;
      }
    };
  }, []);

  // Update clock every second
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  // Clear stale timer so useExamTimer always initializes with the correct duration.
  // Also drop the stale started flags from any previously abandoned exam so the
  // quiz page re-arms the timer fresh instead of inheriting an old start time (F32).
  useEffect(() => {
    localStorage.removeItem("exam_timer");
    localStorage.removeItem("exam_started");
    localStorage.removeItem("exam_started_at");
  }, []);

  // Timer does NOT start until the student clicks Accept
  // Backend stores duration in minutes; convert to seconds for useExamTimer
  const durationSeconds = (data?.duration ?? 60) * 60;
  const timeLeft = useExamTimer(
    durationSeconds,
    "exam_timer",
    () => {
      navigate("/submitted", { replace: true });
    },
    false, // started = false on instructions page
  );

  const handleAccept = () => {
    // Stop camera before navigating to quiz (quiz page will re-request it)
    if (cameraStreamRef.current) {
      cameraStreamRef.current.getTracks().forEach((t) => t.stop());
      cameraStreamRef.current = null;
    }
    localStorage.removeItem("exam_timer");
    navigate("/quiz");
  };

  const handleBackToDashboard = () => {
    if (cameraStreamRef.current) {
      cameraStreamRef.current.getTracks().forEach((t) => t.stop());
      cameraStreamRef.current = null;
    }
    navigate("/dashboard", { replace: true });
  };

  const canBegin = videoProctoring ? cameraStatus === "connected" : true;
  const examDurationHours = durationSeconds / 3600;

  // ---------------------------------------------------------------------------
  // Render: Loading
  // ---------------------------------------------------------------------------

  if (loading) {
    return (
      <div className="flex min-h-screen flex-col bg-background px-4 py-6 sm:px-8">
        <ExamHeader candidateName="" timeLeft={0} />
        <div className="mt-6 flex flex-1 items-center justify-center">
          <div className="flex flex-col items-center">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
            <p className="mt-4 text-sm text-muted-foreground">Loading exam details...</p>
          </div>
        </div>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Render: Error states
  // ---------------------------------------------------------------------------

  if (error) {
    const isAlreadySubmitted = error.includes("already submitted");
    const isNoQuestions = error.includes("no questions") || error.includes("not found");

    return (
      <div className="flex min-h-screen flex-col bg-background px-4 py-6 sm:px-8">
        <ExamHeader candidateName="" timeLeft={0} />
        <div className="mt-6 flex flex-1 items-center justify-center">
          <div className="w-full max-w-md rounded-2xl border border-border bg-card p-8 text-center shadow-sm">
            <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-red-100 ">
              <AlertTriangle className="h-7 w-7 text-red-600 " />
            </div>
            <h2 className="text-base font-semibold text-foreground">
              {isAlreadySubmitted
                ? "Exam Already Submitted"
                : isNoQuestions
                  ? "No Questions Available"
                  : "Unable to Load Exam"}
            </h2>
            <p className="mt-2 text-sm text-muted-foreground">
              {isAlreadySubmitted
                ? "This exam has already been submitted. You cannot retake it."
                : isNoQuestions
                  ? "This exam does not have any questions yet. Please contact your instructor."
                  : error}
            </p>
            <Button onClick={handleBackToDashboard} className="mt-6 gap-2" variant="outline">
              <ArrowLeft className="h-4 w-4" />
              Back to Dashboard
            </Button>
          </div>
        </div>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Render: Instructions
  // ---------------------------------------------------------------------------

  return (
    <div className="flex min-h-screen flex-col bg-background px-4 py-6 sm:px-8">
      <ExamHeader candidateName="" timeLeft={timeLeft} />

      {/* Exam meta info bar */}
      <div className="mt-4 flex flex-wrap items-center gap-4 rounded-2xl border border-border bg-card px-6 py-3 shadow-sm text-sm">
        {/* Test title */}
        {data?.test_title && (
          <>
            <span className="font-semibold text-foreground">{data.test_title}</span>
            <span className="hidden sm:block text-border">|</span>
          </>
        )}

        {/* Exam duration */}
        <div className="flex items-center gap-2 text-foreground font-medium">
          <Clock className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          <span>
            Duration:{" "}
            <span className="font-semibold">
              {examDurationHours === 1
                ? "1 hour"
                : `${examDurationHours} hours`}
            </span>
          </span>
        </div>

        <span className="hidden sm:block text-border">|</span>

        {/* Current date & time */}
        <div className="flex items-center gap-2 text-muted-foreground">
          <CalendarClock className="h-4 w-4" aria-hidden="true" />
          <span>
            {formatCurrentDate(now)}{" "}
            <span className="font-semibold tabular-nums text-foreground">
              {formatCurrentTime(now)}
            </span>
          </span>
        </div>

        {/* Exam date */}
        {data?.exam_date && (
          <>
            <span className="hidden sm:block text-border">|</span>
            <span className="text-muted-foreground">
              Exam Date:{" "}
              <span className="font-semibold text-foreground">{data.exam_date}</span>
            </span>
          </>
        )}
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

          {/* Camera check section — only shown for video proctoring exams */}
          {videoProctoring && (
            <div className="mt-6 rounded-xl border border-border bg-muted/40 p-4">
              <div className="flex items-center gap-2 mb-3">
                <Video className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium text-foreground">Camera Check</span>
              </div>

              {cameraStatus === "checking" && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                  Requesting camera access...
                </div>
              )}

              {cameraStatus === "connected" && (
                <div className="flex flex-col gap-3">
                  <div className="flex items-center gap-2 text-sm text-green-600">
                    <Video className="h-4 w-4" />
                    Camera connected — you may begin
                  </div>
                  <video
                    ref={videoPreviewRef}
                    autoPlay
                    muted
                    playsInline
                    className="w-full max-w-[240px] rounded-lg border bg-black"
                  />
                </div>
              )}

              {cameraStatus === "failed" && (
                <div className="flex flex-col gap-2">
                  <div className="flex items-center gap-2 text-sm text-red-600">
                    <VideoOff className="h-4 w-4" />
                    Camera not available. Please allow camera access and refresh.
                  </div>
                  {cameraError && (
                    <p className="text-xs text-muted-foreground">{cameraError}</p>
                  )}
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={checkCamera}
                    className="mt-1 w-fit gap-1"
                  >
                    <Video className="h-3 w-3" />
                    Retry Camera Check
                  </Button>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer with Accept button */}
        <div className="flex justify-between border-t border-border px-8 py-5">
          <Button variant="outline" onClick={handleBackToDashboard} className="gap-2">
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
          <Button
            size="lg"
            onClick={handleAccept}
            disabled={!canBegin}
            className="min-w-40"
          >
            Accept &amp; Begin
          </Button>
        </div>
      </div>
    </div>
  );
}
