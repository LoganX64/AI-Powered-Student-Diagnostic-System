import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeftIcon, ChevronLeftIcon, ChevronRightIcon, BarChart3Icon } from "lucide-react";
import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
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
import { getStudent, getStudentAssignments, type StudentDetail, type StudentAssignment } from "@/services/admin.service";
import { formatDateDDMMYYYY } from "@/lib/utils";

const PAGE_SIZE = 50;

export function StudentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const studentId = Number(id);

  const [student, setStudent] = useState<StudentDetail | null>(null);
  const [assignments, setAssignments] = useState<StudentAssignment[]>([]);
  const [assignmentTotal, setAssignmentTotal] = useState(0);
  const [assignmentOffset, setAssignmentOffset] = useState(0);

  useEffect(() => {
    if (!id) return;
    getStudent(studentId).then(setStudent).catch(() => {});
  }, [id, studentId]);

  const fetchAssignments = useCallback(async (off: number) => {
    if (!id) return;
    try {
      const res = await getStudentAssignments(studentId);
      setAssignments(res.data ?? []);
      setAssignmentTotal(res.total);
    } catch {
      // silently ignore
    }
  }, [id, studentId]);

  useEffect(() => {
    fetchAssignments(assignmentOffset);
  }, [assignmentOffset, fetchAssignments]);

  if (!student) {
    return (
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset>
          <SiteHeader title="Student Detail" />
          <div className="flex flex-1 items-center justify-center p-6">
            <p className="text-muted-foreground">Loading...</p>
          </div>
        </SidebarInset>
      </SidebarProvider>
    );
  }

  const isDeactivated = student.deleted_at !== null;

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <SiteHeader title={student.name} />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

          {/* Back button */}
          <Button
            variant="ghost"
            size="sm"
            className="w-fit"
            onClick={() => navigate("/admin/students")}
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
                <div>
                  <span className="text-muted-foreground">Coach: </span>
                  <span className="font-medium">{student.coach_name || `#${student.coach_id}`}</span>
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

          {/* SQI Score button */}
          <Button
            variant="outline"
            className="w-fit"
            onClick={() => navigate(`/admin/students/${studentId}/sqi`)}
          >
            <BarChart3Icon className="size-4 mr-2" /> View SQI Score
          </Button>

          {/* Assigned tests table */}
          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">Assigned Tests</h2>
              <Badge variant="secondary">{assignmentTotal}</Badge>
            </div>

            {assignments.length === 0 ? (
              <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                <p className="text-sm text-muted-foreground">No tests assigned to this student.</p>
              </div>
            ) : (
              <>
                <div className="rounded-lg border overflow-hidden">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Test Title</TableHead>
                        <TableHead className="w-36">Status</TableHead>
                        <TableHead>Assigned At</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {assignments.map((a) => (
                        <TableRow key={a.id}>
                          <TableCell className="font-medium">{a.test_title}</TableCell>
                          <TableCell>
                            <Badge variant={a.submitted ? "default" : "secondary"}>
                              {a.submitted ? "Submitted" : "Not Attempted"}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-muted-foreground text-sm">
                            {formatDateDDMMYYYY(a.assigned_at)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                {assignmentTotal > PAGE_SIZE && (
                  <div className="flex items-center justify-between pt-2">
                    <p className="text-sm text-muted-foreground">
                      Showing {assignmentOffset + 1}–{Math.min(assignmentOffset + PAGE_SIZE, assignmentTotal)} of {assignmentTotal}
                    </p>
                    <div className="flex gap-2">
                      <Button variant="outline" size="sm" disabled={assignmentOffset === 0} onClick={() => setAssignmentOffset((o) => Math.max(0, o - PAGE_SIZE))}>
                        <ChevronLeftIcon className="size-4" /> Prev
                      </Button>
                      <Button variant="outline" size="sm" disabled={assignmentOffset + PAGE_SIZE >= assignmentTotal} onClick={() => setAssignmentOffset((o) => o + PAGE_SIZE)}>
                        Next <ChevronRightIcon className="size-4" />
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </div>

        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
