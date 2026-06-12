import { useEffect, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Trash2Icon, ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { CoachSidebar } from "@/components/coach/sidebar";
import { CoachSiteHeader } from "@/components/coach/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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
import { Badge } from "@/components/ui/badge";
import { CreateStudentForm } from "@/components/coach/forms/CreateStudentForm";
import { getStudents, deleteStudent, type Student } from "@/services/coach.service";

const PAGE_SIZE = 50;

export function CoachStudentsPage() {
  const navigate = useNavigate();
  const [students, setStudents] = useState<Student[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [includeDeactivated, setIncludeDeactivated] = useState(false);

  const fetchStudents = useCallback(async (off: number, deactivated: boolean) => {
    try {
      const res = await getStudents({ limit: PAGE_SIZE, offset: off, include_deactivated: deactivated });
      setStudents(res.data ?? []);
      setTotal(res.total);
    } catch {
      // silently ignore
    }
  }, []);

  useEffect(() => {
    fetchStudents(offset, includeDeactivated);
  }, [offset, includeDeactivated, fetchStudents]);

  const handleDelete = async (student: Student) => {
    try {
      setDeletingId(student.student_id);
      await deleteStudent(student.student_id);
      toast.success(`Student "${student.name}" deactivated`);
      fetchStudents(offset, includeDeactivated);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <SidebarProvider>
      <CoachSidebar />
      <SidebarInset>
        <CoachSiteHeader title="Students" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

          <CreateStudentForm
            onCreated={() => fetchStudents(offset, includeDeactivated)}
          />

          <Separator />

          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">All Students</h2>
              <div className="flex items-center gap-3">
                <Button
                  variant={includeDeactivated ? "default" : "outline"}
                  size="sm"
                  onClick={() => {
                    setIncludeDeactivated(!includeDeactivated);
                    setOffset(0);
                  }}
                >
                  {includeDeactivated ? "Showing All" : "Show Deactivated"}
                </Button>
                <Badge variant="secondary">{total}</Badge>
              </div>
            </div>

            {students.length === 0 ? (
              <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                <p className="text-sm text-muted-foreground">
                  No students yet. Create one above.
                </p>
              </div>
            ) : (
              <>
                <div className="rounded-lg border overflow-hidden">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-16">ID</TableHead>
                        <TableHead>Name</TableHead>
                        <TableHead>Code</TableHead>
                        <TableHead className="w-20 text-right">Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {students.map((student) => (
                        <TableRow
                          key={student.student_id}
                          className="cursor-pointer hover:bg-muted/50"
                          onClick={() => navigate(`/coach/students/${student.student_id}`)}
                        >
                          <TableCell className="font-mono text-sm text-muted-foreground">
                            {student.student_id}
                          </TableCell>
                          <TableCell className="font-medium">{student.name}</TableCell>
                          <TableCell>
                            <Badge variant="outline" className="font-mono">
                              {student.student_code}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-right">
                            <AlertDialog>
                              <AlertDialogTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="size-8 text-muted-foreground hover:text-destructive"
                                  disabled={deletingId === student.student_id}
                                  aria-label={`Delete ${student.name}`}
                                  onClick={(e) => e.stopPropagation()}
                                >
                                  <Trash2Icon className="size-4" />
                                </Button>
                              </AlertDialogTrigger>
                              <AlertDialogContent>
                                <AlertDialogHeader>
                                  <AlertDialogTitle>Delete Student</AlertDialogTitle>
                                  <AlertDialogDescription>
                                    Are you sure you want to deactivate{" "}
                                    <span className="font-semibold">{student.name}</span>{" "}
                                    ({student.student_code})?
                                    This action can be reversed by an admin.
                                  </AlertDialogDescription>
                                </AlertDialogHeader>
                                <AlertDialogFooter>
                                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                                  <AlertDialogAction
                                    onClick={(e) => { e.stopPropagation(); handleDelete(student); }}
                                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                  >
                                    Deactivate
                                  </AlertDialogAction>
                                </AlertDialogFooter>
                              </AlertDialogContent>
                            </AlertDialog>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                {total > PAGE_SIZE && (
                  <div className="flex items-center justify-between pt-2">
                    <p className="text-sm text-muted-foreground">
                      Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
                    </p>
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={offset === 0}
                        onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
                      >
                        <ChevronLeftIcon className="size-4" /> Prev
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={offset + PAGE_SIZE >= total}
                        onClick={() => setOffset((o) => o + PAGE_SIZE)}
                      >
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
