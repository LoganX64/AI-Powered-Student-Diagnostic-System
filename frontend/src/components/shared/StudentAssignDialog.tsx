import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SearchableSelect } from "@/components/ui/searchable-select";
import {
  getTests,
  createAssignment,
  type Test,
} from "@/services/dashboard.service";

interface StudentAssignDialogProps {
  studentId: number;
  studentName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAssigned?: () => void;
}

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
  const [tests, setTests] = useState<Test[]>([]);
  const [testId, setTestId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getTests({ limit: 200 })
      .then((res) => setTests(res.data ?? []))
      .catch(() => setTests([]));
  }, []);

  const handleSubmit = async () => {
    setError(null);
    const tid = Number(testId);
    if (!tid) {
      setError("Please select a test");
      return;
    }
    const test = tests.find((t) => t.test_id === tid);
    const coachId = test?.coach_id ?? 0;

    try {
      setSubmitting(true);
      await createAssignment({ student_id: studentId, test_id: tid, coach_id: coachId });
      toast.success(`Assigned "${test?.title ?? "test"}" to ${studentName}`);
      onClose();
      onAssigned?.();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        handleSubmit();
      }}
    >
      <div className="flex flex-col gap-2">
        <Label>Test</Label>
        <SearchableSelect
          options={tests.map((t) => ({
            label: t.title,
            value: t.test_id.toString(),
            search: t.title,
          }))}
          value={testId}
          onChange={setTestId}
          placeholder="Search tests..."
        />
        {error && <p className="text-sm text-destructive">{error}</p>}
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" disabled={submitting}>
          {submitting ? "Assigning…" : "Assign Test"}
        </Button>
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
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Assign Test to {studentName}</DialogTitle>
          <DialogDescription>
            Select a test to assign to this student.
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
