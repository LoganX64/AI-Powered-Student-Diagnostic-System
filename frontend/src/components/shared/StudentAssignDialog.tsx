import { useState, useEffect, useMemo } from "react";
import { toast } from "sonner";
import { CreditCardIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { SearchableSelect } from "@/components/ui/searchable-select";
import { ExamTypePanel } from "@/components/shared/assignment/ExamTypePanel";
import { CostSummary } from "@/components/shared/assignment/CostSummary";
import { EMPTY_POLICY } from "@/components/shared/assignment/exam-presets";
import { computeEstimatedCost } from "@/config/pricing";
import {
  getTests,
  createAssignment,
  type Test,
  type IntegrityPolicy,
} from "@/services/dashboard.service";

interface StudentAssignDialogProps {
  studentId: number;
  studentName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAssigned?: () => void;
}

const STEPS = [
  { n: 1, label: "Select a test" },
  { n: 2, label: "Select exam type" },
  { n: 3, label: "Review & assign" },
] as const;

function AssignForm({
  studentId,
  studentName,
  onClose,
  onAssigned,
}: {
  studentId: number;
  studentName: string;
  onClose: () => void;
  onAssigned?: () => void;
}) {
  const [step, setStep] = useState(1);
  const [tests, setTests] = useState<Test[]>([]);
  const [testId, setTestId] = useState("");
  // Simple by default, matching what this dialog produced before exam types
  // were selectable here.
  const [policy, setPolicy] = useState<IntegrityPolicy>(EMPTY_POLICY);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // has_questions: an empty test must not be assignable — the student would sit a
// timer with nothing to answer.
getTests({ limit: 200, has_questions: true })
      .then((res) => setTests(res.data ?? []))
      .catch((err) =>
        toast.error(
          err instanceof Error ? err.message : "Failed to load tests",
        ),
      );
  }, []);

  const selectedTest = tests.find((t) => t.test_id === Number(testId));
  const durationMin = selectedTest?.duration ?? 0;
  const cost = useMemo(
    () => computeEstimatedCost(policy, durationMin, 1),
    [policy, durationMin],
  );
  const isFinal = step === STEPS.length;

  const goNext = () => {
    setError(null);
    if (step === 1 && !selectedTest) {
      setError("Please select a test");
      return;
    }
    setStep((s) => Math.min(s + 1, STEPS.length));
  };

  const handlePrimary = (e: React.FormEvent) => {
    e.preventDefault();
    if (isFinal) {
      void doCreate();
      return;
    }
    goNext();
  };

  const doCreate = async () => {
    if (!selectedTest) return;
    const coachId = selectedTest.coach_id ?? 0;
    try {
      setSubmitting(true);
      await createAssignment({
        student_id: studentId,
        test_id: selectedTest.test_id,
        coach_id: coachId,
        integrity_policy: policy,
        estimated_cost: cost,
      });
      toast.success(`Assigned "${selectedTest.title}" to ${studentName}`);
      onClose();
      onAssigned?.();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    // No overflow/scroll container here: the test dropdown is absolutely
    // positioned, so any `overflow` ancestor would clip it and grow a
    // scrollbar. DialogContent caps height at 90vh on its own.
    <form
      className="flex flex-col gap-4"
      onSubmit={handlePrimary}
    >
      {/* Progress */}
      <div className="flex flex-col gap-2">
        <div className="flex gap-1.5" aria-hidden="true">
          {STEPS.map((s) => (
            <div
              key={s.n}
              className={`h-1 flex-1 rounded-full ${s.n <= step ? "bg-primary" : "bg-muted"}`}
            />
          ))}
        </div>
        <p className="text-xs text-muted-foreground">
          Step {step} of {STEPS.length} · {STEPS[step - 1].label}
        </p>
      </div>

      {step === 1 && (
        <div className="flex flex-col gap-2">
          <Label>Test</Label>
          <SearchableSelect
            options={tests.map((t) => ({
              label: `${t.title} (${t.duration} min)`,
              value: t.test_id.toString(),
              search: t.title,
            }))}
            value={testId}
            onChange={(v) => {
              setTestId(v);
              setError(null);
            }}
            placeholder="Search tests..."
          />
          {selectedTest && (
            <p className="text-xs text-muted-foreground">
              Duration: {selectedTest.duration} min · Coach: {selectedTest.coach_name}
            </p>
          )}
          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
      )}

      {step === 2 && (
        <Card>
          <CardContent className="flex flex-col gap-3 pt-5">
            <ExamTypePanel policy={policy} onChange={setPolicy} />
          </CardContent>
        </Card>
      )}

      {step === 3 && (
        <div className="flex flex-col gap-4">
          <div className="rounded-md border p-3 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Student</span>
              <span>{studentName}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Test</span>
              <span className="text-right">{selectedTest?.title}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Duration</span>
              <span>{durationMin} min</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Exam type</span>
              <span>
                {policy.server_timing
                  ? policy.video_proctoring
                    ? "Video proctored"
                    : "Backend sync"
                  : "Simple"}
              </span>
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <Label>Final Price</Label>
            <CostSummary policy={policy} durationMin={durationMin} count={1} />
            <p className="text-xs text-muted-foreground">
              This is a simulated payment. No real charge will be made.
            </p>
          </div>
        </div>
      )}

      <DialogFooter>
        {step > 1 && (
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              setError(null);
              setStep((s) => Math.max(s - 1, 1));
            }}
          >
            Back
          </Button>
        )}
        {isFinal ? (
          <Button type="submit" disabled={!selectedTest || submitting}>
            <CreditCardIcon className="size-4" />
            {submitting ? "Assigning…" : `Assign ($${cost.toFixed(2)})`}
          </Button>
        ) : (
          <Button type="submit" disabled={step === 1 && !selectedTest}>
            Next
          </Button>
        )}
      </DialogFooter>
    </form>
  );
}

export function StudentAssignDialog({
  studentId,
  studentName,
  open,
  onOpenChange,
  onAssigned,
}: StudentAssignDialogProps) {
  if (!open) return null;

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Assign Test to {studentName}</DialogTitle>
          <DialogDescription>
            Select a test, choose an exam type, then confirm the price.
          </DialogDescription>
        </DialogHeader>
        <AssignForm
          key={studentId}
          studentId={studentId}
          studentName={studentName}
          onClose={() => onOpenChange(false)}
          onAssigned={onAssigned}
        />
      </DialogContent>
    </Dialog>
  );
}