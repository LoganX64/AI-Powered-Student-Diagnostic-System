import { useEffect, useState, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Trash2Icon, UserPlusIcon, ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
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
import { createStudent, deleteStudent, getStudents, getCoaches, type CreateStudentPayload, type Student, type Coach } from "@/services/dashboard.service";

const PAGE_SIZE = 50;

export function StudentsPage() {
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";
  const isAdmin = role === "admin";

  const [students, setStudents] = useState<Student[]>([]);
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [includeDeactivated, setIncludeDeactivated] = useState(false);

  // Coach search (admin only)
  const [coachSearch, setCoachSearch] = useState("");
  const [selectedCoach, setSelectedCoach] = useState<Coach | null>(null);
  const [coaches, setCoaches] = useState<Coach[]>([]);
  const [showDropdown, setShowDropdown] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [dropdownPos, setDropdownPos] = useState({ top: 0, left: 0, width: 0 });
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();

  const fetchCoaches = useCallback(async (search: string) => {
    if (!isAdmin) return;
    try {
      const res = await getCoaches({ search, limit: 10 });
      setCoaches(res.data ?? []);
    } catch {
      setCoaches([]);
    }
  }, [isAdmin]);

  useEffect(() => {
    if (!isAdmin) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      fetchCoaches(coachSearch);
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [coachSearch, fetchCoaches, isAdmin]);

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

  const handleCoachSelect = (coach: Coach) => {
    setSelectedCoach(coach);
    setCoachSearch(coach.name);
    setShowDropdown(false);
  };

  const handleCoachInputChange = (value: string) => {
    setCoachSearch(value);
    setSelectedCoach(null);
    setShowDropdown(true);
    if (inputRef.current) {
      const rect = inputRef.current.getBoundingClientRect();
      setDropdownPos({ top: rect.bottom + 4, left: rect.left, width: rect.width });
    }
  };

  const handleCreate: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();

    const fd = new FormData(e.currentTarget);

    if (isAdmin && !selectedCoach) {
      toast.error("Please select a coach from the list");
      return;
    }

    const data: CreateStudentPayload = {
      name: fd.get("name") as string,
      student_code: fd.get("student_code") as string,
      coach_id: isAdmin ? selectedCoach!.coach_id : 0,
    };

    try {
      setCreating(true);
      const res = await createStudent(data);
      toast.success(`Student "${data.name}" created — ID: ${res.student_id}`);
      (e.target as HTMLFormElement).reset();
      setCoachSearch("");
      setSelectedCoach(null);
      fetchStudents(offset, includeDeactivated);
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
      toast.success(`Student "${student.name}" ${isAdmin ? "deactivated" : "deleted"}`);
      fetchStudents(offset, includeDeactivated);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <DashboardLayout title="Students">
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
              {isAdmin && (
                <div className="flex flex-col gap-2">
                  <Label htmlFor="student-coach">Coach</Label>
                  <Input
                    ref={inputRef}
                    id="student-coach"
                    placeholder="Search coach by name…"
                    value={coachSearch}
                    onChange={(e) => handleCoachInputChange(e.target.value)}
                    onFocus={() => {
                      setShowDropdown(true);
                      if (inputRef.current) {
                        const rect = inputRef.current.getBoundingClientRect();
                        setDropdownPos({ top: rect.bottom + 4, left: rect.left, width: rect.width });
                      }
                    }}
                    required
                  />
                  {showDropdown && coaches.length > 0 && (
                    <div
                      ref={dropdownRef}
                      style={{ position: "fixed", top: dropdownPos.top, left: dropdownPos.left, width: dropdownPos.width }}
                      className="z-50 max-h-60 overflow-auto rounded-md border bg-popover text-popover-foreground shadow-md"
                    >
                      {coaches.map((coach) => (
                        <button
                          type="button"
                          key={coach.coach_id}
                          className={`flex w-full items-center px-3 py-2 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground ${
                            selectedCoach?.coach_id === coach.coach_id ? "bg-accent text-accent-foreground" : ""
                          }`}
                          onMouseDown={(e) => e.preventDefault()}
                          onClick={() => handleCoachSelect(coach)}
                        >
                          {coach.name}
                        </button>
                      ))}
                    </div>
                  )}
                  {showDropdown && coachSearch && coaches.length === 0 && (
                    <div
                      style={{ position: "fixed", top: dropdownPos.top, left: dropdownPos.left, width: dropdownPos.width }}
                      className="z-50 rounded-md border bg-popover px-3 py-2 text-sm text-muted-foreground shadow-md"
                    >
                      No coaches found
                    </div>
                  )}
                </div>
              )}
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
          <div className="rounded-lg border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Code</TableHead>
                  {isAdmin && <TableHead className="w-24">Coach ID</TableHead>}
                  <TableHead className="w-20 text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {students.map((student) => (
                  <TableRow
                    key={student.student_id}
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => navigate(`${prefix}/students/${student.student_id}`)}
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
                        {student.coach_id}
                      </TableCell>
                    )}
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
                              Are you sure you want to {isAdmin ? "delete" : "deactivate"}{" "}
                              <span className="font-semibold">{student.name}</span>{" "}
                              ({student.student_code})?
                              {isAdmin ? " This action cannot be undone." : " This action can be reversed by an admin."}
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={(e) => { e.stopPropagation(); handleDelete(student); }}
                              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                            >
                              {isAdmin ? "Delete" : "Deactivate"}
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

        {/* Pagination */}
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
      </div>
    </DashboardLayout>
  );
}
