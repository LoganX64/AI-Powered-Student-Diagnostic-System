import { useEffect, useState, useCallback } from "react";
import { toast } from "sonner";
import { Trash2Icon, BookOpenIcon } from "lucide-react";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Pagination, PaginationContent, PaginationItem, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
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
import { createSubject, deleteSubject, getSubjects, reactivateSubject, type Subject } from "@/services/dashboard.service";
import { createSubjectSchema, zodErrors } from "@/lib/validations";

const PAGE_SIZE = 50;

export function SubjectsPage() {
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [creating, setCreating] = useState(false);
  const [createErrors, setCreateErrors] = useState<Record<string, string>>({});
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [reactivateTarget, setReactivateTarget] = useState<{ id: number; name: string } | null>(null);

  const fetchSubjects = useCallback(async (off: number) => {
    try {
      const res = await getSubjects({ limit: PAGE_SIZE, offset: off });
      setSubjects(res.data ?? []);
      setTotal(res.total);
    } catch {
      // silently ignore
    }
  }, []);

  useEffect(() => {
    fetchSubjects(offset);
  }, [offset, fetchSubjects]);

  const handleCreate: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const raw = { name: fd.get("name") as string };

    const result = createSubjectSchema.safeParse(raw);
    if (!result.success) {
      setCreateErrors(zodErrors(result.error));
      return;
    }
    setCreateErrors({});

    try {
      setCreating(true);
      const res = await createSubject(result.data);
      toast.success(`Subject "${result.data.name}" created — ID: ${res.subject_id}`);
      (e.target as HTMLFormElement).reset();
      fetchSubjects(offset);
    } catch (err) {
      const e = err as Error & { payload?: { deactivated_id?: number; deactivated_name?: string } };
      if (e.payload?.deactivated_id) {
        setReactivateTarget({ id: e.payload.deactivated_id, name: e.payload.deactivated_name ?? "" });
      } else {
        toast.error(e.message);
      }
    } finally {
      setCreating(false);
    }
  };

  const handleReactivate = async () => {
    if (!reactivateTarget) return;
    try {
      await reactivateSubject(reactivateTarget.id);
      toast.success(`Subject "${reactivateTarget.name}" reactivated`);
      setReactivateTarget(null);
      fetchSubjects(offset);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const handleDelete = async (subject: Subject) => {
    try {
      setDeletingId(subject.subject_id);
      await deleteSubject(subject.subject_id);
      toast.success(`Subject "${subject.name}" deactivated`);
      fetchSubjects(offset);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <DashboardLayout title="Subjects">
      {/* Create form */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BookOpenIcon className="size-5" />
            Create Subject
          </CardTitle>
          <CardDescription>
            Add a new subject to your organization.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreate} className="flex flex-col gap-4">
            <div className="flex gap-3 items-end">
              <div className="flex flex-col gap-2 max-w-sm w-full">
                <Label htmlFor="subject-name">Subject Name</Label>
                <Input
                  id="subject-name"
                  name="name"
                  placeholder="Mathematics"
                  required
                />
                {createErrors.name && <p className="text-sm text-destructive">{createErrors.name}</p>}
              </div>
              <Button type="submit" disabled={creating}>
                {creating ? "Creating…" : "Create Subject"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Separator />

      {/* Subjects table */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">All Subjects</h2>
          <Badge variant="secondary">{total}</Badge>
        </div>

        {subjects.length === 0 ? (
          <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">
              No subjects yet. Create one above.
            </p>
          </div>
        ) : (
          <div className="rounded-lg border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead className="w-20 text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {subjects.map((subject) => (
                  <TableRow key={subject.subject_id}>
                    <TableCell className="font-mono text-sm text-muted-foreground">
                      {subject.subject_id}
                    </TableCell>
                    <TableCell className="font-medium">{subject.name}</TableCell>
                    <TableCell className="text-right">
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="size-8 text-muted-foreground hover:text-destructive"
                            disabled={deletingId === subject.subject_id}
                            aria-label={`Delete ${subject.name}`}
                          >
                            <Trash2Icon className="size-4" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>Delete Subject</AlertDialogTitle>
                            <AlertDialogDescription>
                              Are you sure you want to deactivate{" "}
                              <span className="font-semibold">{subject.name}</span>?
                              This subject will be deactivated. It can be reactivated later.
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => handleDelete(subject)}
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

      {/* Reactivation prompt */}
      <AlertDialog open={reactivateTarget !== null} onOpenChange={(open) => { if (!open) setReactivateTarget(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Reactivate Subject</AlertDialogTitle>
            <AlertDialogDescription>
              Subject <span className="font-semibold">"{reactivateTarget?.name}"</span> already exists but is deactivated.
              Would you like to reactivate it?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setReactivateTarget(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleReactivate}>
              Reactivate
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </DashboardLayout>
  );
}
