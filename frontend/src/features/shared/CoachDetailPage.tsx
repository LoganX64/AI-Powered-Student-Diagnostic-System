import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeftIcon } from "lucide-react";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Pagination, PaginationContent, PaginationItem, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
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
  getCoach,
  getCoachTests,
  getCoachStudents,
  type CoachDetail,
  type CoachTest,
  type CoachStudent,
} from "@/services/dashboard.service";
import { formatDateDDMMYYYY, parseRouteId } from "@/lib/utils";

const PAGE_SIZE = 50;

export function CoachDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const coachId = parseRouteId(id);

  const [coach, setCoach] = useState<CoachDetail | null>(null);
  const [tests, setTests] = useState<CoachTest[]>([]);
  const [testTotal, setTestTotal] = useState(0);
  const [testOffset, setTestOffset] = useState(0);
  const [students, setStudents] = useState<CoachStudent[]>([]);
  const [studentTotal, setStudentTotal] = useState(0);
  const [studentOffset, setStudentOffset] = useState(0);

  useEffect(() => {
    if (coachId === null) return;
    getCoach(coachId).then(setCoach).catch(() => {});
  }, [coachId]);

  const fetchTests = useCallback(async (off: number) => {
    if (coachId === null) return;
    try {
      const res = await getCoachTests(coachId, { limit: PAGE_SIZE, offset: off });
      setTests(res.data ?? []);
      setTestTotal(res.total);
    } catch {
      // silently ignore
    }
  }, [coachId]);

  const fetchStudents = useCallback(async (off: number) => {
    if (coachId === null) return;
    try {
      const res = await getCoachStudents(coachId, { limit: PAGE_SIZE, offset: off });
      setStudents(res.data ?? []);
      setStudentTotal(res.total);
    } catch {
      // silently ignore
    }
  }, [coachId]);

  useEffect(() => {
    fetchTests(testOffset);
  }, [testOffset, fetchTests]);

  useEffect(() => {
    fetchStudents(studentOffset);
  }, [studentOffset, fetchStudents]);

  if (coachId === null) {
    return (
      <DashboardLayout title="Coach Not Found">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Invalid coach ID in URL.</p>
        </div>
      </DashboardLayout>
    );
  }

  if (!coach) {
    return (
      <DashboardLayout title="Coach Detail">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout title={coach.name}>
      {/* Back button */}
      <Button
        variant="ghost"
        size="sm"
        className="w-fit"
        onClick={() => navigate("/admin/coaches")}
      >
        <ArrowLeftIcon className="size-4 mr-2" /> Back to Coaches
      </Button>

      {/* Coach info */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-3 flex-wrap">
            {coach.name}
            <Badge variant="outline" className="font-mono">ID: {coach.coach_id}</Badge>
            <Badge variant="outline" className="font-mono">{coach.email}</Badge>
            {coach.deleted_at && (
              <Badge variant="destructive">Deactivated</Badge>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-4 text-sm">
            <div>
              <span className="text-muted-foreground">User ID: </span>
              <span className="font-medium">{coach.user_id}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Created: </span>
              <span className="font-medium">{formatDateDDMMYYYY(coach.created_at)}</span>
            </div>
            {coach.deleted_at && (
              <div>
                <span className="text-muted-foreground">Deactivated: </span>
                <span className="font-medium">{formatDateDDMMYYYY(coach.deleted_at)}</span>
                {coach.deleted_by_name && (
                  <>
                    <span className="text-muted-foreground"> by </span>
                    <span className="font-medium">{coach.deleted_by_name}</span>
                  </>
                )}
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Students section */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">Students</h2>
          <Badge variant="secondary">{studentTotal}</Badge>
        </div>

        {students.length === 0 ? (
          <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">No students assigned to this coach.</p>
          </div>
        ) : (
          <>
            <div className="rounded-lg border overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-16">ID</TableHead>
                    <TableHead>Name</TableHead>
                    <TableHead>Student Code</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {students.map((s) => (
                    <TableRow
                      key={s.student_id}
                      className="cursor-pointer hover:bg-muted/50"
                      onClick={() => navigate(`/admin/students/${s.student_id}`)}
                    >
                      <TableCell className="font-mono text-sm text-muted-foreground">
                        {s.student_id}
                      </TableCell>
                      <TableCell className="font-medium">{s.name}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="font-mono">{s.student_code}</Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {formatDateDDMMYYYY(s.created_at)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {studentTotal > PAGE_SIZE && (
              <Pagination>
                <PaginationContent className="flex items-center justify-between w-full">
                  <p className="text-sm text-muted-foreground">
                    Showing {studentOffset + 1}–{Math.min(studentOffset + PAGE_SIZE, studentTotal)} of {studentTotal}
                  </p>
                  <div className="flex gap-2">
                    <PaginationItem>
                      <PaginationPrevious
                        onClick={() => setStudentOffset((o) => Math.max(0, o - PAGE_SIZE))}
                        className={studentOffset === 0 ? "pointer-events-none opacity-50" : ""}
                      />
                    </PaginationItem>
                    <PaginationItem>
                      <PaginationNext
                        onClick={() => setStudentOffset((o) => o + PAGE_SIZE)}
                        className={studentOffset + PAGE_SIZE >= studentTotal ? "pointer-events-none opacity-50" : ""}
                      />
                    </PaginationItem>
                  </div>
                </PaginationContent>
              </Pagination>
            )}
          </>
        )}
      </div>

      {/* Tests / Question Papers section */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">Question Papers</h2>
          <Badge variant="secondary">{testTotal}</Badge>
        </div>

        {tests.length === 0 ? (
          <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">No question papers created by this coach.</p>
          </div>
        ) : (
          <>
            <div className="rounded-lg border overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-16">ID</TableHead>
                    <TableHead>Title</TableHead>
                    <TableHead>Subject</TableHead>
                    <TableHead className="w-24">Duration</TableHead>
                    <TableHead>Exam Date</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tests.map((t) => (
                    <TableRow
                      key={t.test_id}
                      className="cursor-pointer hover:bg-muted/50"
                      onClick={() => navigate(`/admin/tests/${t.test_id}`)}
                    >
                      <TableCell className="font-mono text-sm text-muted-foreground">
                        {t.test_id}
                      </TableCell>
                      <TableCell className="font-medium">{t.title}</TableCell>
                      <TableCell>{t.subject_name || `#${t.subject_id}`}</TableCell>
                      <TableCell>{t.duration} min</TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {t.exam_date ? formatDateDDMMYYYY(t.exam_date) : "—"}
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {formatDateDDMMYYYY(t.created_at)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {testTotal > PAGE_SIZE && (
              <Pagination>
                <PaginationContent className="flex items-center justify-between w-full">
                  <p className="text-sm text-muted-foreground">
                    Showing {testOffset + 1}–{Math.min(testOffset + PAGE_SIZE, testTotal)} of {testTotal}
                  </p>
                  <div className="flex gap-2">
                    <PaginationItem>
                      <PaginationPrevious
                        onClick={() => setTestOffset((o) => Math.max(0, o - PAGE_SIZE))}
                        className={testOffset === 0 ? "pointer-events-none opacity-50" : ""}
                      />
                    </PaginationItem>
                    <PaginationItem>
                      <PaginationNext
                        onClick={() => setTestOffset((o) => o + PAGE_SIZE)}
                        className={testOffset + PAGE_SIZE >= testTotal ? "pointer-events-none opacity-50" : ""}
                      />
                    </PaginationItem>
                  </div>
                </PaginationContent>
              </Pagination>
            )}
          </>
        )}
      </div>
    </DashboardLayout>
  );
}
