import { useEffect, useState, useCallback } from "react";
import { toast } from "sonner";
import { Trash2Icon, BookOpenIcon, ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
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
import { createSubject, deleteSubject, getSubjects, type Subject } from "@/services/admin.service";

const PAGE_SIZE = 50;

export function SubjectsPage() {
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);

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
    const name = fd.get("name") as string;

    try {
      setCreating(true);
      const res = await createSubject({ name });
      toast.success(`Subject "${name}" created — ID: ${res.subject_id}`);
      (e.target as HTMLFormElement).reset();
      fetchSubjects(offset);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (subject: Subject) => {
    try {
      setDeletingId(subject.subject_id);
      await deleteSubject(subject.subject_id);
      toast.success(`Subject "${subject.name}" deleted`);
      fetchSubjects(offset);
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
        <SiteHeader title="Subjects" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

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
                                  Are you sure you want to delete{" "}
                                  <span className="font-semibold">{subject.name}</span>?
                                  This action cannot be undone.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() => handleDelete(subject)}
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

        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
