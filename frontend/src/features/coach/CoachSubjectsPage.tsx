import { useState } from "react";
import { toast } from "sonner";
import { Trash2Icon } from "lucide-react";
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
import { CreateSubjectForm, type CoachSubject } from "@/components/coach/forms/CreateSubjectForm";

export function CoachSubjectsPage() {
  const [subjects, setSubjects] = useState<CoachSubject[]>([]);

  const handleDelete = (subject: CoachSubject) => {
    setSubjects((prev) => prev.filter((s) => s.subject_id !== subject.subject_id));
    toast.success(`Subject "${subject.name}" deleted`);
  };

  return (
    <SidebarProvider>
      <CoachSidebar />
      <SidebarInset>
        <CoachSiteHeader title="Subjects" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

          <CreateSubjectForm
            onCreated={(subject) => setSubjects((prev) => [subject, ...prev])}
          />

          <Separator />

          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">All Subjects</h2>
              <Badge variant="secondary">{subjects.length}</Badge>
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
          </div>

        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
