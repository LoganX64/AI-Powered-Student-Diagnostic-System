import { useCallback, useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeftIcon, BarChart3Icon, BarChartIcon, PencilIcon, Trash2Icon } from "lucide-react";
import { toast } from "sonner";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  getStudent,
  getStudentAssignments,
  getBatches,
  deleteAssignment,
  type StudentDetail,
  type StudentAssignment,
  type Batch,
} from "@/services/dashboard.service";
import { formatDateDDMMYYYY, parseRouteId } from "@/lib/utils";
import { StudentFormDialog } from "@/components/shared/StudentFormDialog";
import { StudentAssignDialog } from "@/components/shared/StudentAssignDialog";
import { LiveVideoPanel } from "@/components/shared/LiveVideoPanel";

export function StudentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";
  const studentId = parseRouteId(id);

  const [student, setStudent] = useState<StudentDetail | null>(null);
  const [assignments, setAssignments] = useState<StudentAssignment[]>([]);
  const [assignmentTotal, setAssignmentTotal] = useState(0);
  const [batches, setBatches] = useState<Batch[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [assignDialogOpen, setAssignDialogOpen] = useState(false);
  const [assignFilter, setAssignFilter] = useState<"active" | "all">("active");
  const [assignOffset, setAssignOffset] = useState(0);

  const PAGE_SIZE = 50;

  const fetchAssignments = useCallback(
    async (off: number, filter: "active" | "all") => {
      if (studentId == null) return;
      try {
        const res = await getStudentAssignments(studentId, {
          limit: PAGE_SIZE,
          offset: off,
          status: filter === "active" ? "active" : undefined,
        });
        setAssignments(res.data ?? []);
        setAssignmentTotal(res.total);
      } catch {
        /* keep previous list on refetch error */
      }
    },
    [studentId]
  );

  useEffect(() => {
    if (studentId === null) return;
    getStudent(studentId).then(setStudent).catch(() => {});
  }, [studentId]);

  useEffect(() => {
    getBatches()
      .then((res) => setBatches(res.data ?? []))
      .catch(() => setBatches([]));
  }, []);

  useEffect(() => {
    if (studentId == null) return;
    getStudentAssignments(studentId, {
      limit: PAGE_SIZE,
      offset: assignOffset,
      status: assignFilter === "active" ? "active" : undefined,
    })
      .then((res) => {
        setAssignments(res.data ?? []);
        setAssignmentTotal(res.total);
      })
      .catch(() => {});
  }, [studentId, assignFilter, assignOffset]);

  if (studentId === null) {
    return (
      <DashboardLayout title="Student Not Found">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Invalid student ID in URL.</p>
        </div>
      </DashboardLayout>
    );
  }

  if (!student) {
    return (
      <DashboardLayout title="Student Detail">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </DashboardLayout>
    );
  }

  const isDeactivated = student.deleted_at !== null;

  return (
    <DashboardLayout key={studentId ?? "none"} title={student.name}>
      {/* Back button */}
      <Button
        variant="ghost"
        size="sm"
        className="w-fit"
        onClick={() => navigate(`${prefix}/students`)}
      >
        <ArrowLeftIcon className="size-4 mr-2" /> Back to Students
      </Button>

      {/* Student info */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-3 flex-wrap">
            {student.name}
            <Badge variant="outline" className="font-mono">ID: {student.student_id}</Badge>
            <Badge variant="outline" className="font-mono">{student.student_code}</Badge>
            {isDeactivated && (
              <Badge variant="destructive">Deactivated</Badge>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-4 text-sm">
            {role === "admin" && (
              <div>
                <span className="text-muted-foreground">Coach: </span>
                <span className="font-medium">{student.coach_name || `#${student.coach_id}`}</span>
              </div>
            )}
            <div>
              <span className="text-muted-foreground">Batch: </span>
              <span className="font-medium">
                {student.batch_id != null
                  ? (batches.find((b) => b.id === student.batch_id)?.name ?? `Batch #${student.batch_id}`)
                  : "—"}
              </span>
            </div>
            <div>
              <span className="text-muted-foreground">Created: </span>
              <span className="font-medium">{formatDateDDMMYYYY(student.created_at)}</span>
            </div>
            {isDeactivated && student.deleted_by_name && (
              <div>
                <span className="text-muted-foreground">Deactivated by: </span>
                <span className="font-medium">
                  {student.deleted_by_name} ({student.deleted_by_role})
                  {" on "}
                  {student.deleted_at && formatDateDDMMYYYY(student.deleted_at)}
                </span>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Actions */}
      <div className="flex flex-wrap gap-3">
        <Button
          variant="outline"
          className="w-fit"
          onClick={() => setAssignDialogOpen(true)}
        >
          <BarChart3Icon className="size-4 mr-2" /> Assign Test
        </Button>
        <Button
          variant="outline"
          className="w-fit"
          onClick={() => setDialogOpen(true)}
        >
          <PencilIcon className="size-4 mr-2" /> Edit Student
        </Button>
        <Button
          variant="outline"
          className="w-fit"
          onClick={() => navigate(`${prefix}/students/${studentId}/sqi`)}
        >
          <BarChart3Icon className="size-4 mr-2" /> View SQI Score
        </Button>
      </div>

      {/* Live Video Preview (shown when student has active exam in progress) */}
      {assignments.some((a) => a.attempt_in_progress) && (
        <LiveVideoPanel studentId={studentId} studentName={student.name} />
      )}

      {/* Assigned tests table */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">Assigned Tests</h2>
          <div className="flex items-center gap-2">
            <div className="flex rounded-md border p-0.5 text-sm">
              <button
                type="button"
                onClick={() => { setAssignOffset(0); setAssignFilter("active"); }}
                className={`rounded px-2.5 py-1 ${assignFilter === "active" ? "bg-secondary font-medium" : "text-muted-foreground"}`}
              >
                Active
              </button>
              <button
                type="button"
                onClick={() => { setAssignOffset(0); setAssignFilter("all"); }}
                className={`rounded px-2.5 py-1 ${assignFilter === "all" ? "bg-secondary font-medium" : "text-muted-foreground"}`}
              >
                All
              </button>
            </div>
            <Badge variant="secondary">{assignmentTotal}</Badge>
          </div>
        </div>

        {(() => {
          if (assignments.length === 0) {
            return (
              <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                <p className="text-sm text-muted-foreground">
                  {assignFilter === "active"
                    ? "No active (unsubmitted) tests assigned to this student."
                    : "No tests assigned to this student."}
                </p>
              </div>
            );
          }
          return (
            <>
              <div className="rounded-lg border overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Test Title</TableHead>
                      <TableHead className="w-36">Status</TableHead>
                      <TableHead>Assigned At</TableHead>
                      <TableHead className="w-40 text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {assignments.map((a) => (
                      <TableRow
                        key={a.id}
                        className="cursor-pointer hover:bg-muted/50"
                        onClick={() => navigate(`${prefix}/students/${studentId}/assignments/${a.id}`)}
                      >
                        <TableCell className="font-medium">{a.test_title}</TableCell>
                        <TableCell>
                          <Badge variant={a.submitted ? "default" : "secondary"}>
                            {a.submitted ? "Submitted" : "Active"}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground text-sm">
                          {formatDateDDMMYYYY(a.assigned_at)}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center justify-end gap-1" onClick={(e) => e.stopPropagation()}>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="gap-1"
                              onClick={() => navigate(`${prefix}/students/${studentId}/sqi`)}
                            >
                              <BarChartIcon className="size-3.5" /> SQI
                            </Button>
                            <AlertDialog>
                              <AlertDialogTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="size-8 text-muted-foreground hover:text-destructive"
                                  aria-label={`Cancel assignment ${a.test_title}`}
                                >
                                  <Trash2Icon className="size-4" />
                                </Button>
                              </AlertDialogTrigger>
                              <AlertDialogContent>
                                <AlertDialogHeader>
                                  <AlertDialogTitle>Cancel Assigned Test</AlertDialogTitle>
                                  <AlertDialogDescription>
                                    Are you sure you want to cancel the assignment of{" "}
                                    <span className="font-semibold">{a.test_title}</span>{" "}
                                    for {student.name}? This cannot be undone.
                                  </AlertDialogDescription>
                                </AlertDialogHeader>
                                <AlertDialogFooter>
                                  <AlertDialogCancel>Keep Assignment</AlertDialogCancel>
                                  <AlertDialogAction
                                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                    onClick={async () => {
                                      try {
                                        await deleteAssignment(a.id);
                                        toast.success("Assignment cancelled");
                                        fetchAssignments(assignOffset, assignFilter);
                                      } catch (err) {
                                        toast.error((err as Error).message);
                                      }
                                    }}
                                  >
                                    Cancel Test
                                  </AlertDialogAction>
                                </AlertDialogFooter>
                              </AlertDialogContent>
                            </AlertDialog>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              {assignmentTotal > PAGE_SIZE && (
                <Pagination>
                  <PaginationContent className="flex items-center justify-between w-full">
                    <p className="text-sm text-muted-foreground">
                      Showing {assignOffset + 1}–{Math.min(assignOffset + PAGE_SIZE, assignmentTotal)} of {assignmentTotal}
                    </p>
                    <div className="flex gap-2">
                      <PaginationItem>
                        <PaginationPrevious
                          onClick={() => setAssignOffset((o) => Math.max(0, o - PAGE_SIZE))}
                          className={assignOffset === 0 ? "pointer-events-none opacity-50" : ""}
                        />
                      </PaginationItem>
                      <PaginationItem>
                        <PaginationNext
                          onClick={() => setAssignOffset((o) => o + PAGE_SIZE)}
                          className={assignOffset + PAGE_SIZE >= assignmentTotal ? "pointer-events-none opacity-50" : ""}
                        />
                      </PaginationItem>
                    </div>
                  </PaginationContent>
                </Pagination>
              )}
            </>
          );
        })()}
      </div>

      <StudentFormDialog
        mode="edit"
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        studentId={studentId ?? undefined}
        initial={
          student
            ? {
                name: student.name,
                student_code: student.student_code,
                coach_id: student.coach_id,
                batch_id: student.batch_id,
                coach_name: student.coach_name,
              }
            : undefined
        }
        onSaved={() => {
          if (studentId != null) {
            getStudent(studentId).then(setStudent).catch(() => {});
          }
        }}
      />

      <StudentAssignDialog
        studentId={studentId ?? 0}
        studentName={student?.name ?? ""}
        open={assignDialogOpen}
        onOpenChange={setAssignDialogOpen}
        onAssigned={() => {
          setAssignOffset(0);
          setAssignFilter("active");
          fetchAssignments(0, "active");
        }}
      />
    </DashboardLayout>
  );
}
