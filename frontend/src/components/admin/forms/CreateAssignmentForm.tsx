import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import {
  ClipboardListIcon,
  FolderIcon,
  ServerIcon,
  UsersIcon,
  VideoIcon,
  CreditCardIcon,
} from "lucide-react";
import { useRole } from "@/hooks/useRole";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
    <div className="flex flex-col gap-6">
      {/* Step 1: Test */}
      <Card>
        <CardHeader>
          <CardTitle>1. Select Test</CardTitle>
          <CardDescription>Choose the diagnostic test to assign.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <Label>Test</Label>
          <Select value={selectedTestId} onValueChange={setSelectedTestId}>
            <SelectTrigger>
              <SelectValue placeholder="Select a test" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {tests.map((t) => (
                  <SelectItem key={t.test_id} value={t.test_id.toString()}>
                    {t.title} ({t.duration} min)
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          {selectedTest && (
            <p className="text-xs text-muted-foreground">
              Duration: {selectedTest.duration} min · Coach: {selectedTest.coach_name}
            </p>
          )}
        </CardContent>
      </Card>

      {/* Step 2: Target */}
      <Card>
        <CardHeader>
          <CardTitle>2. Select Target</CardTitle>
          <CardDescription>
            Assign to a single student or an entire batch. The live count drives pricing.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex gap-2">
            <Button
              type="button"
              variant={targetType === "student" ? "default" : "outline"}
              size="sm"
              onClick={() => setTargetType("student")}
            >
              <UsersIcon className="size-4" /> Single student
            </Button>
            <Button
              type="button"
              variant={targetType === "batch" ? "default" : "outline"}
              size="sm"
              onClick={() => setTargetType("batch")}
            >
              <FolderIcon className="size-4" /> Batch
            </Button>
          </div>

          {targetType === "student" ? (
            <div className="flex flex-col gap-2">
              <Label>Student</Label>
              <Select
                value={selectedStudentId}
                onValueChange={setSelectedStudentId}
                disabled={!selectedTest}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a student" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {eligibleStudents.map((s) => (
                      <SelectItem key={s.student_id} value={s.student_id.toString()}>
                        {s.name} ({s.student_code})
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              {selectedTest && eligibleStudents.length === 0 && (
                <p className="text-xs text-destructive">
                  No students belong to this test&apos;s coach.
                </p>
              )}
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              <Label>Batch</Label>
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
                Only students in the batch who belong to this test&apos;s coach are counted.
              </p>
            </div>
          )}

          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Students to assign:</span>
            <Badge variant="secondary">{count}</Badge>
          </div>
        </CardContent>
      </Card>

      {/* Step 3: Exam type */}
      <Card>
        <CardHeader>
          <CardTitle>3. Exam Type &amp; Integrity</CardTitle>
          <CardDescription>
            Pick a preset, or fine-tune the individual integrity flags below.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid gap-3 sm:grid-cols-3">
            {PRESETS.map((p) => {
              const active = samePolicy(policy, presetPolicy(p.key));
              const Icon = p.icon;
              return (
                <button
                  type="button"
                  key={p.key}
                  onClick={() => setPolicy(presetPolicy(p.key))}
                  className={`flex flex-col gap-2 rounded-lg border p-4 text-left transition-colors ${
                    active
                      ? "border-primary bg-primary/5"
                      : "hover:bg-accent hover:text-accent-foreground"
                  }`}
                >
                  <Icon className="size-5" />
                  <span className="font-medium">{p.title}</span>
                  <span className="text-xs text-muted-foreground">{p.desc}</span>
                </button>
              );
            })}
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="flex items-center justify-between rounded-md border p-3">
              <div>
                <p className="text-sm font-medium">Server timing</p>
                <p className="text-xs text-muted-foreground">Authoritative start/deadline</p>
              </div>
              <Switch
                checked={policy.server_timing}
                onCheckedChange={(v) => setPolicy((p) => ({ ...p, server_timing: v }))}
              />
            </div>
            <div className="flex items-center justify-between rounded-md border p-3">
              <div>
                <p className="text-sm font-medium">Autosave</p>
                <p className="text-xs text-muted-foreground">Server-side answer backups</p>
              </div>
              <Switch
                checked={policy.autosave}
                onCheckedChange={(v) => setPolicy((p) => ({ ...p, autosave: v }))}
              />
            </div>
            <div className="flex items-center justify-between rounded-md border p-3">
              <div>
                <p className="text-sm font-medium">Tab switch detection</p>
                <p className="text-xs text-muted-foreground">Log visibility changes</p>
              </div>
              <Switch
                checked={policy.tab_switch_detect}
                onCheckedChange={(v) => setPolicy((p) => ({ ...p, tab_switch_detect: v }))}
              />
            </div>
            <div className="flex items-center justify-between rounded-md border p-3">
              <div>
                <p className="text-sm font-medium">Video proctoring</p>
                <p className="text-xs text-muted-foreground">Record-only chunks</p>
              </div>
              <Switch
                checked={policy.video_proctoring}
                onCheckedChange={(v) => setPolicy((p) => ({ ...p, video_proctoring: v }))}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Step 4: Live price + pay */}
      <Card>
        <CardHeader>
          <CardTitle>4. Review &amp; Pay</CardTitle>
          <CardDescription>Estimated cost updates live with flags and student count.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="rounded-md border text-sm">
            <div className="flex justify-between px-4 py-2">
              <span className="text-muted-foreground">Base ({count} × ${PRICING.base_rate_per_student})</span>
              <span>${base.toFixed(2)}</span>
            </div>
            {timing > 0 && (
              <div className="flex justify-between px-4 py-2">
                <span className="text-muted-foreground">Server timing (flat)</span>
                <span>+${timing.toFixed(2)}</span>
              </div>
            )}
            {autosave > 0 && (
              <div className="flex justify-between px-4 py-2">
                <span className="text-muted-foreground">Autosave (flat)</span>
                <span>+${autosave.toFixed(2)}</span>
              </div>
            )}
            {tab > 0 && (
              <div className="flex justify-between px-4 py-2">
                <span className="text-muted-foreground">Tab detection (flat)</span>
                <span>+${tab.toFixed(2)}</span>
              </div>
            )}
            {video > 0 && (
              <div className="flex justify-between px-4 py-2">
                <span className="text-muted-foreground">
                  Video (${PRICING.video_rate_per_student_min}/std-min × {durationMin} × {count})
                </span>
                <span>+${video.toFixed(2)}</span>
              </div>
            )}
            <div className="flex justify-between border-t px-4 py-2 font-semibold">
              <span>Total estimated cost</span>
              <span>${cost.toFixed(2)}</span>
            </div>
          </div>

          <Button onClick={() => setPayOpen(true)} disabled={!canProceed} className="w-fit">
            <CreditCardIcon className="size-4" />
            Proceed to Payment (${cost.toFixed(2)})
          </Button>
          {!canProceed && (
            <p className="text-xs text-muted-foreground">
              Select a test and at least one eligible student to continue.
            </p>
          )}
        </CardContent>
      </Card>

      <Dialog open={payOpen} onOpenChange={setPayOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Mock Payment</DialogTitle>
            <DialogDescription>
              This is a simulated payment. No real charge will be made.
            </DialogDescription>
          </DialogHeader>
          <div className="rounded-md border p-4 text-sm">
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
