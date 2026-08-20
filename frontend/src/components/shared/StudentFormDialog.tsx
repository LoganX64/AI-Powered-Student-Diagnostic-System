import { useState, useEffect, useRef } from "react";
import { toast } from "sonner";
import { useRole } from "@/hooks/useRole";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
import {
  createStudent,
  updateStudent,
  getCoaches,
  getBatches,
  type Coach,
  type Batch,
} from "@/services/dashboard.service";
import { createStudentSchema, updateStudentSchema, zodErrors } from "@/lib/validations";

export type StudentFormInitial = {
  name: string;
  student_code: string;
  coach_id: number;
  batch_id: number | null;
  coach_name?: string;
};

interface StudentFormDialogProps {
  mode: "create" | "edit";
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initial?: StudentFormInitial;
  studentId?: number;
  onSaved?: () => void;
}

function StudentFormFields({
  mode,
  initial,
  studentId,
  onSaved,
  onClose,
  isAdmin,
}: {
  mode: "create" | "edit";
  initial?: StudentFormInitial;
  studentId?: number;
  onSaved?: () => void;
  onClose: () => void;
  isAdmin: boolean;
}) {
  const [name, setName] = useState(initial?.name ?? "");
  const [studentCode, setStudentCode] = useState(initial?.student_code ?? "");
  const [selectedBatchId, setSelectedBatchId] = useState(
    initial?.batch_id != null ? String(initial.batch_id) : "none"
  );

  const [coachSearch, setCoachSearch] = useState("");
  const [selectedCoach, setSelectedCoach] = useState<Coach | null>(null);
  const [coaches, setCoaches] = useState<Coach[]>([]);
  const [showDropdown, setShowDropdown] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const prefilledRef = useRef(false);

  const [batches, setBatches] = useState<Batch[]>([]);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    getBatches()
      .then((res) => setBatches(res.data ?? []))
      .catch(() => setBatches([]));
  }, []);

  // Loads coaches and, on first load in edit mode, prefills the already-assigned coach.
  const initialCoachId = initial?.coach_id;
  const initialCoachName = initial?.coach_name;
  useEffect(() => {
    if (!isAdmin) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(async () => {
      try {
        const res = await getCoaches({ search: coachSearch, limit: 200 });
        const list = res.data ?? [];
        setCoaches(list);
        if (mode === "edit" && !prefilledRef.current && initialCoachId) {
          const match = list.find((c) => c.coach_id === initialCoachId);
          if (match) {
            setSelectedCoach(match);
            setCoachSearch(match.name);
          } else {
            setCoachSearch(initialCoachName || `Coach #${initialCoachId}`);
          }
          prefilledRef.current = true;
        }
      } catch {
        setCoaches([]);
      }
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [coachSearch, isAdmin, mode, initialCoachId, initialCoachName]);

  useEffect(() => {
    if (!isAdmin) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isAdmin]);

  const handleSelectCoach = (coach: Coach) => {
    setSelectedCoach(coach);
    setCoachSearch(coach.name);
    setShowDropdown(false);
  };

  const handleCoachInputChange = (value: string) => {
    setCoachSearch(value);
    setSelectedCoach(null);
    setShowDropdown(true);
  };

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();

    const coachId = isAdmin ? selectedCoach?.coach_id ?? initial?.coach_id ?? 0 : 0;
    const batchId = selectedBatchId === "none" ? null : Number(selectedBatchId);

    const raw = {
      name: name.trim(),
      student_code: studentCode.trim() || undefined,
      coach_id: isAdmin ? (coachId || undefined) : undefined,
      batch_id: batchId,
    };

    const schema = mode === "edit" ? updateStudentSchema : createStudentSchema;
    const result = schema.safeParse(raw);
    if (!result.success) {
      setErrors(zodErrors(result.error));
      return;
    }
    setErrors({});

    try {
      setSubmitting(true);
      if (mode === "edit" && studentId != null) {
        await updateStudent(studentId, result.data);
        toast.success(`Student "${result.data.name}" updated`);
      } else {
        await createStudent(result.data);
        toast.success(`Student "${result.data.name}" created`);
      }
      onClose();
      onSaved?.();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <Label htmlFor="student-name">Full Name</Label>
        <Input
          id="student-name"
          name="name"
          placeholder="Alice"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="student-code">Student Code</Label>
        <Input
          id="student-code"
          name="student_code"
          placeholder="Auto-generated if blank"
          value={studentCode}
          onChange={(e) => setStudentCode(e.target.value)}
        />
        <p className="text-xs text-muted-foreground">
          Leave blank to auto-generate a unique code.
        </p>
        {errors.student_code && (
          <p className="text-sm text-destructive">{errors.student_code}</p>
        )}
      </div>

      {isAdmin && (
        <div className="flex flex-col gap-2 relative" ref={dropdownRef}>
          <Label htmlFor="student-coach">Coach</Label>
          <Input
            id="student-coach"
            placeholder="Search coach by name…"
            value={coachSearch}
            onChange={(e) => handleCoachInputChange(e.target.value)}
            onFocus={() => setShowDropdown(true)}
          />
          {showDropdown && coaches.length > 0 && (
            <div className="absolute top-full left-0 right-0 z-50 mt-1 max-h-60 overflow-auto rounded-md border bg-popover text-popover-foreground shadow-md">
              {coaches.map((coach) => (
                <button
                  type="button"
                  key={coach.coach_id}
                  className={`flex w-full items-center px-3 py-2 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground ${
                    selectedCoach?.coach_id === coach.coach_id ? "bg-accent text-accent-foreground" : ""
                  }`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => handleSelectCoach(coach)}
                >
                  {coach.name}
                </button>
              ))}
            </div>
          )}
          {showDropdown && coachSearch && coaches.length === 0 && (
            <div className="absolute top-full left-0 right-0 z-50 mt-1 rounded-md border bg-popover px-3 py-2 text-sm text-muted-foreground shadow-md">
              No coaches found
            </div>
          )}
          {errors.coach_id && <p className="text-sm text-destructive">{errors.coach_id}</p>}
        </div>
      )}

      <div className="flex flex-col gap-2">
        <Label htmlFor="student-batch">Batch (optional)</Label>
        <Select value={selectedBatchId} onValueChange={setSelectedBatchId}>
          <SelectTrigger id="student-batch">
            <SelectValue placeholder="No batch" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="none">No batch</SelectItem>
              {batches.map((b) => (
                <SelectItem key={b.id} value={b.id.toString()}>
                  {b.name}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" disabled={submitting}>
          {submitting
            ? mode === "edit"
              ? "Saving…"
              : "Creating…"
            : mode === "edit"
              ? "Save Changes"
              : "Create Student"}
        </Button>
      </DialogFooter>
    </form>
  );
}

export function StudentFormDialog({
  mode,
  open,
  onOpenChange,
  initial,
  studentId,
  onSaved,
}: StudentFormDialogProps) {
  const role = useRole();
  if (!open) return null;

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{mode === "edit" ? "Edit Student" : "Add Student"}</DialogTitle>
          <DialogDescription>
            {mode === "edit"
              ? "Update the student's details, coach, and batch assignment."
              : "Add a new student and assign them to a coach and batch (optional)."}
          </DialogDescription>
        </DialogHeader>
        <StudentFormFields
          key={studentId ?? "create"}
          mode={mode}
          initial={initial}
          studentId={studentId}
          onSaved={onSaved}
          onClose={() => onOpenChange(false)}
          isAdmin={role === "admin"}
        />
      </DialogContent>
    </Dialog>
  );
}
