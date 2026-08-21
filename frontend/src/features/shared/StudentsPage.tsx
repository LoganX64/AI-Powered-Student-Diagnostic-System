import { useEffect, useState, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Trash2Icon, UserPlusIcon, RotateCcwIcon, SearchIcon, PencilIcon } from "lucide-react";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
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
import { Separator } from "@/components/ui/separator";
import { Pagination, PaginationContent, PaginationItem, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
import { deleteStudent, reactivateStudent, getStudents, type Student } from "@/services/dashboard.service";
import { StudentFormDialog } from "@/components/shared/StudentFormDialog";

const PAGE_SIZE = 50;

export function StudentsPage() {
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";
  const isAdmin = role === "admin";

  const [students, setStudents] = useState<Student[]>([]);
  const [deactivatingId, setDeactivatingId] = useState<number | null>(null);
  const [reactivatingId, setReactivatingId] = useState<number | null>(null);
  const [dialogOpenId, setDialogOpenId] = useState<number | null>(null);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [includeDeactivated, setIncludeDeactivated] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);

  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const searchDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  // Dialog (create / edit)
  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogMode, setDialogMode] = useState<"create" | "edit">("create");
  const [editingStudent, setEditingStudent] = useState<Student | null>(null);

  useEffect(() => {
    if (searchDebounceRef.current) clearTimeout(searchDebounceRef.current);
    searchDebounceRef.current = setTimeout(() => {
      setSearch(searchInput);
      setOffset(0);
    }, 300);
    return () => { if (searchDebounceRef.current) clearTimeout(searchDebounceRef.current); };
  }, [searchInput]);

  const fetchStudents = useCallback(async (off: number, deactivated: boolean, searchTerm: string) => {
    try {
      const res = await getStudents({ limit: PAGE_SIZE, offset: off, include_deactivated: deactivated, search: searchTerm || undefined });
      setStudents(res.data ?? []);
      setTotal(res.total);
      setFetchError(null);
    } catch (err) {
      const message = (err as Error).message || "Failed to load students";
      setFetchError(message);
      toast.error(message);
    }
  }, []);

  useEffect(() => {
    fetchStudents(offset, includeDeactivated, search);
  }, [offset, includeDeactivated, search, fetchStudents]);

  const openCreate = () => {
    setDialogMode("create");
    setEditingStudent(null);
    setDialogOpen(true);
  };

  const openEdit = (student: Student) => {
    setDialogMode("edit");
    setEditingStudent(student);
    setDialogOpen(true);
  };

  const handleDeactivate = async (student: Student) => {
    try {
      setDeactivatingId(student.student_id);
      await deleteStudent(student.student_id);
      toast.success(`Student "${student.name}" account deactivated`);
      fetchStudents(offset, includeDeactivated, search);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setDeactivatingId(null);
    }
  };

  const handleReactivate = async (student: Student) => {
    try {
      setReactivatingId(student.student_id);
      await reactivateStudent(student.student_id);
      toast.success(`Student "${student.name}" account reactivated`);
      fetchStudents(offset, includeDeactivated, search);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setReactivatingId(null);
    }
  };

  const studentInitial = editingStudent
    ? {
        name: editingStudent.name,
        student_code: editingStudent.student_code,
        coach_id: editingStudent.coach_id,
        batch_id: editingStudent.batch_id ?? null,
      }
    : undefined;

  return (
    <DashboardLayout title="Students">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">Students</h1>
          <p className="text-sm text-muted-foreground">
            Manage student accounts and their batch assignments.
          </p>
        </div>
        <Button onClick={openCreate} className="w-fit">
          <UserPlusIcon className="size-4 mr-2" /> Add Student
        </Button>
      </div>

      <Separator />

      {/* Students table */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">All Students</h2>
          <div className="flex items-center gap-3">
            <div className="relative flex-1 max-w-sm">
              <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
              <input
                placeholder="Search by name or code..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                className="flex h-9 w-full rounded-md border border-input bg-transparent pl-9 pr-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              />
            </div>
            <Button
              variant={includeDeactivated ? "default" : "outline"}
              size="sm"
              onClick={() => {
                setIncludeDeactivated(!includeDeactivated);
                setOffset(0);
              }}
            >
              {includeDeactivated ? "Showing Deactivated" : "Show Deactivated"}
            </Button>
            <Badge variant="secondary">{total}</Badge>
          </div>
        </div>

        {fetchError ? (
          <div className="flex flex-col gap-4">
            <div className="flex h-32 items-center justify-center rounded-lg border border-destructive/50 bg-destructive/10">
              <p className="text-sm text-destructive">{fetchError}</p>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="w-fit"
              onClick={() => fetchStudents(offset, includeDeactivated, search)}
            >
              Retry
            </Button>
          </div>
        ) : students.length === 0 ? (
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
                  {isAdmin && <TableHead className="w-32">Coach</TableHead>}
                  <TableHead className="w-28 text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {students.map((student) => (
                  <TableRow
                    key={student.student_id}
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => {
                      if (dialogOpenId !== student.student_id) {
                        navigate(`${prefix}/students/${student.student_id}`);
                      }
                    }}
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
                    {isAdmin && (
                      <TableCell className="text-muted-foreground">
                        {student.coach_name || "—"}
                      </TableCell>
                    )}
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1" onClick={(e) => e.stopPropagation()}>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-8 text-muted-foreground hover:text-foreground"
                          aria-label={`Edit ${student.name}`}
                          onClick={() => openEdit(student)}
                        >
                          <PencilIcon className="size-4" />
                        </Button>
                        {student.deleted_at ? (
                          <AlertDialog onOpenChange={(open) => setDialogOpenId(open ? student.student_id : null)}>
                            <AlertDialogTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="size-8 text-muted-foreground hover:text-green-600"
                                disabled={reactivatingId === student.student_id}
                                aria-label={`Reactivate account for ${student.name}`}
                              >
                                <RotateCcwIcon className="size-4" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Reactivate Account</AlertDialogTitle>
                                <AlertDialogDescription>
                                  Are you sure you want to reactivate the account for{" "}
                                  <span className="font-semibold">{student.name}</span>{" "}
                                  ({student.student_code})?
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={(e) => { e.stopPropagation(); handleReactivate(student); }}
                                  className="bg-green-600 text-white hover:bg-green-700"
                                >
                                  Reactivate Account
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        ) : (
                          <AlertDialog onOpenChange={(open) => setDialogOpenId(open ? student.student_id : null)}>
                            <AlertDialogTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="size-8 text-muted-foreground hover:text-destructive"
                                disabled={deactivatingId === student.student_id}
                                aria-label={`Deactivate account for ${student.name}`}
                              >
                                <Trash2Icon className="size-4" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Deactivate Account</AlertDialogTitle>
                                <AlertDialogDescription>
                                  Are you sure you want to deactivate the account for{" "}
                                  <span className="font-semibold">{student.name}</span>{" "}
                                  ({student.student_code})?
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={(e) => { e.stopPropagation(); handleDeactivate(student); }}
                                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                >
                                  Deactivate Account
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        {/* Pagination */}
        {total > PAGE_SIZE && (
          <Pagination>
            <PaginationContent className="flex items-center justify-between w-full">
              <p className="text-sm text-muted-foreground">
                Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
              </p>
              <div className="flex gap-2">
                <PaginationItem>
                  <PaginationPrevious
                    onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
                    className={offset === 0 ? "pointer-events-none opacity-50" : ""}
                  />
                </PaginationItem>
                <PaginationItem>
                  <PaginationNext
                    onClick={() => setOffset((o) => o + PAGE_SIZE)}
                    className={offset + PAGE_SIZE >= total ? "pointer-events-none opacity-50" : ""}
                  />
                </PaginationItem>
              </div>
            </PaginationContent>
          </Pagination>
        )}
      </div>

      <StudentFormDialog
        mode={dialogMode}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        initial={studentInitial}
        studentId={editingStudent?.student_id}
        onSaved={() => fetchStudents(offset, includeDeactivated, search)}
      />
    </DashboardLayout>
  );
}
