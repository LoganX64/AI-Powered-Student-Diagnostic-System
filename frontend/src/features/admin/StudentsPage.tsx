import { useEffect, useState, useCallback } from "react";
import { toast } from "sonner";
import { Trash2Icon, UserPlusIcon, ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
import { createStudent, deleteStudent, getStudents, type CreateStudentPayload, type Student } from "@/services/admin.service";

const PAGE_SIZE = 50;

export function StudentsPage() {
  const [students, setStudents] = useState<Student[]>([]);
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);

  const fetchStudents = useCallback(async (off: number) => {
    try {
      const res = await getStudents({ limit: PAGE_SIZE, offset: off });
      setStudents(res.data);
      setTotal(res.total);
    } catch {
      // silently ignore
    }
  }, []);

  useEffect(() => {
    fetchStudents(offset);
  }, [offset, fetchStudents]);

  const handleCreate: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const coachId = Number(fd.get("coach_id"));

    if (!coachId || coachId < 1) {
      toast.error("Coach ID must be a valid positive number");
      return;
    }

    const data: CreateStudentPayload = {
      name: fd.get("name") as string,
      student_code: fd.get("student_code") as string,
      coach_id: coachId,
    };

    try {
      setCreating(true);
      const res = await createStudent(data);
      const newStudent: Student = {
        student_id: res.student_id,
        name: data.name,
        student_code: data.student_code,
        coach_id: data.coach_id,
      };
      setStudents((prev) => [newStudent, ...prev]);
      toast.success(`Student "${data.name}" created — ID: ${res.student_id}`);
      (e.target as HTMLFormElement).reset();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (student: Student) => {
    try {
      setDeletingId(student.student_id);
      await deleteStudent(student.student_id);
      setStudents((prev) => prev.filter((s) => s.student_id !== student.student_id));
      toast.success(`Student "${student.name}" deleted`);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <SiteHeader title="Students" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

          {/* Create form */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <UserPlusIcon className="size-5" />
                Create Student
              </CardTitle>
              <CardDescription>
                Add a new student and assign them to a coach.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleCreate} className="flex flex-col gap-4">
                <div className="grid gap-4 sm:grid-cols-3">
                  <div className="flex flex-col gap-2">
                    <Label htmlFor="student-name">Full Name</Label>
                    <Input
                      id="student-name"
                      name="name"
                      placeholder="Alice"
                      required
                    />
                  </div>
                  <div className="flex flex-col gap-2">
                    <Label htmlFor="student-code">Student Code</Label>
                    <Input
                      id="student-code"
                      name="student_code"
                      placeholder="STU001"
                      required
                    />
                  </div>
                  <div className="flex flex-col gap-2">
                    <Label htmlFor="student-coach-id">Coach ID</Label>
                    <Input
                      id="student-coach-id"
                      name="coach_id"
                      type="number"
                      min={1}
                      placeholder="1"
                      required
                    />
                  </div>
                </div>
                <Button type="submit" disabled={creating} className="w-fit">
                  {creating ? "Creating…" : "Create Student"}
                </Button>
              </form>
            </CardContent>
          </Card>

          <Separator />

          {/* Students table */}
          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">All Students</h2>
              <Badge variant="secondary">{students.length}</Badge>
            </div>

            {students.length === 0 ? (
              <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                <p className="text-sm text-muted-foreground">
                  No students yet. Create one above.
                </p>
              </div>
            ) : (
              <div className="rounded-lg border overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-16">ID</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Code</TableHead>
                      <TableHead className="w-24">Coach ID</TableHead>
                      <TableHead className="w-20 text-right">Action</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {students.map((student) => (
                      <TableRow key={student.student_id}>
                        <TableCell className="font-mono text-sm text-muted-foreground">
                          {student.student_id}
                        </TableCell>
                        <TableCell className="font-medium">{student.name}</TableCell>
                        <TableCell>
                          <Badge variant="outline" className="font-mono">
                            {student.student_code}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {student.coach_id}
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
                              >
                                <Trash2Icon className="size-4" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Delete Student</AlertDialogTitle>
                                <AlertDialogDescription>
                                  Are you sure you want to delete{" "}
                                  <span className="font-semibold">{student.name}</span>{" "}
                                  ({student.student_code})?
                                  This action cannot be undone.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() => handleDelete(student)}
                                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                >
                                  Delete
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
            )}
          </div>

        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
