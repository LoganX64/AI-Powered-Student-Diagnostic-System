import { useEffect, useMemo, useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import {
  ClipboardListIcon,
  FolderIcon,
  ServerIcon,
  UsersIcon,
  VideoIcon,
  CreditCardIcon,
  ChevronDownIcon,
  CheckIcon,
  SearchIcon,
} from "lucide-react";
import { useRole } from "@/hooks/useRole";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  getTests,
  getStudents,
  getBatches,
  createAssignment,
  createBatchAssignment,
  type Test,
  type Student,
  type Batch,
  type IntegrityPolicy,
} from "@/services/dashboard.service";
import { computeEstimatedCost, PRICING } from "@/config/pricing";

const EMPTY_POLICY: IntegrityPolicy = {
  server_timing: false,
  autosave: false,
  video_proctoring: false,
  tab_switch_detect: false,
};

type ExamPreset = "simple" | "backend" | "video";

function presetPolicy(preset: ExamPreset): IntegrityPolicy {
  switch (preset) {
    case "backend":
      return { server_timing: true, autosave: true, video_proctoring: false, tab_switch_detect: true };
    case "video":
      return { server_timing: true, autosave: true, video_proctoring: true, tab_switch_detect: true };
    default:
      return { ...EMPTY_POLICY };
  }
}

const PRESETS: { key: ExamPreset; title: string; icon: typeof ClipboardListIcon; desc: string }[] = [
  {
    key: "simple",
    title: "Simple",
    icon: ClipboardListIcon,
    desc: "Client-only timing. No server sync, autosave, or video.",
  },
  {
    key: "backend",
    title: "Backend sync",
    icon: ServerIcon,
    desc: "Server-authoritative timing + autosave + tab detection.",
  },
  {
    key: "video",
    title: "Video proctored",
    icon: VideoIcon,
    desc: "Everything in Backend sync, plus video recording.",
  },
];

function samePolicy(a: IntegrityPolicy, b: IntegrityPolicy): boolean {
  return (
    a.server_timing === b.server_timing &&
    a.autosave === b.autosave &&
    a.video_proctoring === b.video_proctoring &&
    a.tab_switch_detect === b.tab_switch_detect
  );
}

function StudentPicker({
  students,
  value,
  onChange,
  disabled,
}: {
  students: Student[];
  value: string;
  onChange: (id: string) => void;
  disabled: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const selected = students.find((s) => String(s.student_id) === value);

  const displayValue = open ? search : selected ? `${selected.name} (${selected.student_code})` : "";

  const filtered = students
    .filter(
      (s) =>
        s.name.toLowerCase().includes(search.toLowerCase()) ||
        s.student_code.toLowerCase().includes(search.toLowerCase())
    )
    .slice(0, 50);

  const totalMatches = students.filter(
    (s) =>
      s.name.toLowerCase().includes(search.toLowerCase()) ||
      s.student_code.toLowerCase().includes(search.toLowerCase())
  ).length;

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
        setSearch("");
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <div className="relative" ref={containerRef}>
      <div className="relative">
        <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <input
          ref={inputRef}
          type="text"
          placeholder="Search student by name or code..."
          value={displayValue}
          onChange={(e) => {
            setSearch(e.target.value);
            if (!open) setOpen(true);
          }}
          onFocus={() => {
            setOpen(true);
            setSearch("");
          }}
          disabled={disabled}
          className="flex h-9 w-full rounded-md border bg-transparent pl-9 pr-8 py-2 text-sm transition-colors placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
        />
        <ChevronDownIcon className="absolute right-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
      </div>

      {open && (
        <div className="absolute z-50 mt-1 max-h-48 w-full overflow-y-auto rounded-md border bg-popover p-1 shadow-sm">
          {filtered.length === 0 ? (
            <p className="py-2 text-center text-xs text-muted-foreground">No students found.</p>
          ) : (
            filtered.map((s) => (
              <button
                key={s.student_id}
                type="button"
                onClick={() => {
                  onChange(String(s.student_id));
                  setOpen(false);
                  setSearch("");
                }}
                className={`flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground ${
                  value === String(s.student_id) ? "bg-accent" : ""
                }`}
              >
                <CheckIcon className={`size-3.5 shrink-0 ${value === String(s.student_id) ? "opacity-100" : "opacity-0"}`} />
                <span>{s.name}</span>
                <span className="ml-auto text-xs text-muted-foreground font-mono">{s.student_code}</span>
              </button>
            ))
          )}
          {totalMatches > 50 && (
            <p className="py-1.5 text-center text-xs text-muted-foreground">
              Showing 50 of {totalMatches} — type more to narrow
            </p>
          )}
        </div>
      )}
    </div>
  );
}

export function CreateAssignmentForm() {
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";
  const navigate = useNavigate();

  const [tests, setTests] = useState<Test[]>([]);
  const [students, setStudents] = useState<Student[]>([]);
  const [batches, setBatches] = useState<Batch[]>([]);
  const [loading, setLoading] = useState(true);

  const [selectedTestId, setSelectedTestId] = useState("");
  const [targetType, setTargetType] = useState<"student" | "batch">("student");
  const [selectedStudentId, setSelectedStudentId] = useState("");
  const [selectedBatchId, setSelectedBatchId] = useState("");
  const [policy, setPolicy] = useState<IntegrityPolicy>(EMPTY_POLICY);

  const [payOpen, setPayOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const [t, s, b] = await Promise.all([
          getTests({ limit: 10000 }),
          getStudents({ limit: 10000 }),
          getBatches(),
        ]);
        if (!active) return;
        setTests(t.data ?? []);
        setStudents(s.data ?? []);
        setBatches(b.data ?? []);
      } catch (err) {
        toast.error((err as Error).message);
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  const selectedTest = useMemo(
    () => tests.find((t) => String(t.test_id) === selectedTestId),
    [tests, selectedTestId]
  );
  const coachId = selectedTest?.coach_id ?? 0;
  const durationMin = selectedTest?.duration ?? 0;

  const eligibleStudents = useMemo(
    () => students.filter((s) => s.coach_id === coachId && !s.deleted_at),
    [students, coachId]
  );

  const targetStudentIds = useMemo(() => {
    if (!selectedTest) return [];
    if (targetType === "student") {
      return selectedStudentId ? [Number(selectedStudentId)] : [];
    }
    if (!selectedBatchId) return [];
    return students
      .filter(
        (s) =>
          s.batch_id === Number(selectedBatchId) &&
          s.coach_id === coachId &&
          !s.deleted_at
      )
      .map((s) => s.student_id);
  }, [students, targetType, selectedStudentId, selectedBatchId, coachId, selectedTest]);

  const count = targetStudentIds.length;
  const cost = useMemo(
    () => computeEstimatedCost(policy, durationMin, count),
    [policy, durationMin, count]
  );

  const base = PRICING.base_rate_per_student * count;
  const timing = policy.server_timing ? PRICING.timing_flat : 0;
  const autosave = policy.autosave ? PRICING.autosave_flat : 0;
  const tab = policy.tab_switch_detect ? PRICING.tab_flat : 0;
  const video = policy.video_proctoring
    ? PRICING.video_rate_per_student_min * durationMin * count
    : 0;

  const canProceed = !!selectedTest && count > 0 && !submitting;

  const resetAfter = () => {
    setSelectedTestId("");
    setTargetType("student");
    setSelectedStudentId("");
    setSelectedBatchId("");
    setPolicy(EMPTY_POLICY);
  };

  const doCreate = async () => {
    if (!selectedTest || count === 0) return;
    setSubmitting(true);
    try {
      if (targetType === "student" && count === 1) {
        const res = await createAssignment({
          student_id: targetStudentIds[0],
          test_id: selectedTest.test_id,
          coach_id: coachId,
          integrity_policy: policy,
          estimated_cost: cost,
        });
        toast.success(`Test assigned — Assignment ID: ${res.assignment_id}`);
      } else {
        const res = await createBatchAssignment({
          test_id: selectedTest.test_id,
          student_ids: targetStudentIds,
          coach_id: coachId,
          integrity_policy: policy,
          estimated_cost: cost,
        });
        toast.success(`${res.created} assignment(s) created`);
      }
      setPayOpen(false);
      resetAfter();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <Card>
        <CardContent className="py-12 text-center text-sm text-muted-foreground">
          Loading…
        </CardContent>
      </Card>
    );
  }

  if (tests.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center gap-3 py-12">
          <ClipboardListIcon className="size-8 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No tests created yet.</p>
          <Button variant="outline" size="sm" onClick={() => navigate(`${prefix}/tests`)}>
            Create a Test
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Row 1: Test + Target side by side */}
      <div className="grid gap-4 lg:grid-cols-2">
        {/* Test selection */}
        <Card>
          <CardContent className="flex flex-col gap-2 pt-5">
            <Label>Test</Label>
            <SearchableSelect
              options={tests.map((t) => ({
                label: `${t.title} (${t.duration} min)`,
                value: t.test_id.toString(),
                search: t.title,
              }))}
              value={selectedTestId}
              onChange={setSelectedTestId}
              placeholder="Search tests..."
            />
            {selectedTest && (
              <p className="text-xs text-muted-foreground">
                Duration: {selectedTest.duration} min · Coach: {selectedTest.coach_name}
              </p>
            )}
          </CardContent>
        </Card>

        {/* Target selection */}
        <Card className="overflow-visible">
          <CardContent className="flex flex-col gap-2 pt-5">
            <div className="flex items-center justify-between">
              <Label>Assign to</Label>
              <div className="flex gap-1">
                <button
                  type="button"
                  onClick={() => setTargetType("student")}
                  className={`flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                    targetType === "student"
                      ? "bg-primary text-primary-foreground"
                      : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
                  }`}
                >
                  <UsersIcon className="size-3" /> Student
                </button>
                <button
                  type="button"
                  onClick={() => setTargetType("batch")}
                  className={`flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
                    targetType === "batch"
                      ? "bg-primary text-primary-foreground"
                      : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
                  }`}
                >
                  <FolderIcon className="size-3" /> Batch
                </button>
              </div>
            </div>

            {targetType === "student" ? (
              <>
                <StudentPicker
                  students={eligibleStudents}
                  value={selectedStudentId}
                  onChange={setSelectedStudentId}
                  disabled={!selectedTest}
                />
                {selectedTest && eligibleStudents.length === 0 && (
                  <p className="text-xs text-destructive">
                    No students belong to this test&apos;s coach.
                  </p>
                )}
              </>
            ) : (
              <>
                <Select
                  value={selectedBatchId}
                  onValueChange={setSelectedBatchId}
                  disabled={!selectedTest}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select a batch" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {batches.map((b) => (
                        <SelectItem key={b.id} value={b.id.toString()}>
                          {b.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Students in this batch matching the test&apos;s coach.
                </p>
              </>
            )}

            <div className="flex items-center gap-2 pt-1">
              <span className="text-xs text-muted-foreground">Students to assign:</span>
              <Badge variant="secondary" className="text-xs">{count}</Badge>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Row 2: Exam type + Pricing side by side */}
      <div className="grid gap-4 lg:grid-cols-3">
        {/* Exam type */}
        <Card className="lg:col-span-2">
          <CardContent className="flex flex-col gap-3 pt-5">
            <Label>Exam Type &amp; Integrity</Label>
            <div className="grid gap-2 sm:grid-cols-3">
              {PRESETS.map((p) => {
                const active = samePolicy(policy, presetPolicy(p.key));
                const Icon = p.icon;
                return (
                  <button
                    type="button"
                    key={p.key}
                    onClick={() => setPolicy(presetPolicy(p.key))}
                    className={`flex items-start gap-3 rounded-lg border p-3 text-left transition-colors ${
                      active
                        ? "border-primary bg-primary/5"
                        : "hover:bg-accent hover:text-accent-foreground"
                    }`}
                  >
                    <Icon className="size-4 mt-0.5 shrink-0" />
                    <div>
                      <span className="text-sm font-medium">{p.title}</span>
                      <span className="block text-xs text-muted-foreground">{p.desc}</span>
                    </div>
                  </button>
                );
              })}
            </div>

            <div className="grid gap-2 sm:grid-cols-2">
              {[
                { label: "Server timing", desc: "Authoritative start/deadline", key: "server_timing" as const },
                { label: "Autosave", desc: "Server-side answer backups", key: "autosave" as const },
                { label: "Tab switch detection", desc: "Log visibility changes", key: "tab_switch_detect" as const },
                { label: "Video proctoring", desc: "Record-only chunks", key: "video_proctoring" as const },
              ].map((item) => (
                <div key={item.key} className="flex items-center justify-between rounded-md border px-3 py-2">
                  <div>
                    <p className="text-sm font-medium">{item.label}</p>
                    <p className="text-xs text-muted-foreground">{item.desc}</p>
                  </div>
                  <Switch
                    checked={policy[item.key]}
                    onCheckedChange={(v) => setPolicy((p) => ({ ...p, [item.key]: v }))}
                  />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Pricing + Submit */}
        <Card>
          <CardContent className="flex flex-col gap-3 pt-5">
            <Label>Estimated Cost</Label>
            <div className="rounded-md border text-sm">
              <div className="flex justify-between px-3 py-1.5">
                <span className="text-muted-foreground">Base ({count} × ${PRICING.base_rate_per_student})</span>
                <span>${base.toFixed(2)}</span>
              </div>
              {timing > 0 && (
                <div className="flex justify-between px-3 py-1.5">
                  <span className="text-muted-foreground">Timing</span>
                  <span>+${timing.toFixed(2)}</span>
                </div>
              )}
              {autosave > 0 && (
                <div className="flex justify-between px-3 py-1.5">
                  <span className="text-muted-foreground">Autosave</span>
                  <span>+${autosave.toFixed(2)}</span>
                </div>
              )}
              {tab > 0 && (
                <div className="flex justify-between px-3 py-1.5">
                  <span className="text-muted-foreground">Tab detect</span>
                  <span>+${tab.toFixed(2)}</span>
                </div>
              )}
              {video > 0 && (
                <div className="flex justify-between px-3 py-1.5">
                  <span className="text-muted-foreground">Video</span>
                  <span>+${video.toFixed(2)}</span>
                </div>
              )}
              <div className="flex justify-between border-t px-3 py-2 font-semibold">
                <span>Total</span>
                <span>${cost.toFixed(2)}</span>
              </div>
            </div>

            <Button onClick={() => setPayOpen(true)} disabled={!canProceed} className="w-full">
              <CreditCardIcon className="size-4" />
              Assign (${cost.toFixed(2)})
            </Button>
            {!canProceed && (
              <p className="text-xs text-muted-foreground text-center">
                Select a test and at least one student to continue.
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      <Dialog open={payOpen} onOpenChange={setPayOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Assignment</DialogTitle>
            <DialogDescription>
              This is a simulated payment. No real charge will be made.
            </DialogDescription>
          </DialogHeader>
          <div className="rounded-md border p-3 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Test</span>
              <span>{selectedTest?.title}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Students</span>
              <span>{count}</span>
            </div>
            <div className="flex justify-between font-semibold">
              <span>Amount</span>
              <span>${cost.toFixed(2)}</span>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPayOpen(false)}>
              Cancel
            </Button>
            <Button onClick={doCreate} disabled={submitting}>
              {submitting ? "Processing…" : "Pay (mock)"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
