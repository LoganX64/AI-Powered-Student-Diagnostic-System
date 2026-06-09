import { useState } from "react";
import { toast } from "sonner";
import { Trash2Icon } from "lucide-react";
import { CoachSidebar } from "@/components/coach/sidebar";
import { CoachSiteHeader } from "@/components/coach/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import { ClipboardListIcon, LinkIcon } from "lucide-react";
import { CreateTestForm, type CoachTest } from "@/components/coach/forms/CreateTestForm";
import { CreateQuestionsForm } from "@/components/coach/forms/CreateQuestionsForm";
import { CreateAssignmentForm, type CoachAssignment } from "@/components/coach/forms/CreateAssignmentForm";

const TABS = [
  { value: "test", label: "Create Test & Questions", icon: ClipboardListIcon },
  { value: "assign", label: "Assign", icon: LinkIcon },
];

export function CoachTestsPage() {
  const [tests, setTests] = useState<CoachTest[]>([]);
  const [createdTestId, setCreatedTestId] = useState<number | null>(null);
  const [assignments, setAssignments] = useState<CoachAssignment[]>([]);

  return (
    <SidebarProvider>
      <CoachSidebar />
      <SidebarInset>
        <CoachSiteHeader title="Tests" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

          <div>
            <h2 className="text-lg font-semibold">Tests &amp; Assignments</h2>
            <p className="text-sm text-muted-foreground">
              Create a test, add questions to it, then assign it to a student.
            </p>
          </div>

          <Tabs defaultValue="test" className="w-full">
            <TabsList className="mb-2">
              {TABS.map(({ value, label, icon: Icon }) => (
                <TabsTrigger key={value} value={value} className="flex items-center gap-2">
                  <Icon className="size-4" />
                  {label}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent value="test" className="flex flex-col gap-6">
              {createdTestId === null ? (
                <CreateTestForm
                  onCreated={(test) => {
                    setTests((prev) => [test, ...prev]);
                    setCreatedTestId(test.test_id);
                  }}
                />
              ) : (
                <>
                  <div className="flex items-center gap-3 rounded-lg border border-dashed p-4">
                    <span className="text-sm text-muted-foreground">
                      Test created with ID <span className="font-mono font-semibold text-foreground">{createdTestId}</span>. Now add questions below.
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setCreatedTestId(null);
                        toast.info("Ready to create a new test");
                      }}
                    >
                      Create Another Test
                    </Button>
                  </div>
                  <CreateQuestionsForm
                    testId={createdTestId}
                    onCreated={(id, count) => {
                      toast.success(`${count} question(s) added to test ${id}`);
                    }}
                  />
                </>
              )}

              {tests.length > 0 && (
                <div className="flex flex-col gap-3">
                  <div className="flex items-center justify-between">
                    <h3 className="text-base font-semibold">Created Tests</h3>
                    <Badge variant="secondary">{tests.length}</Badge>
                  </div>
                  <div className="rounded-lg border overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-16">ID</TableHead>
                          <TableHead>Title</TableHead>
                          <TableHead>Subject ID</TableHead>
                          <TableHead>Duration</TableHead>
                          <TableHead className="w-20 text-right">Action</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {tests.map((test) => (
                          <TableRow key={test.test_id}>
                            <TableCell className="font-mono text-sm text-muted-foreground">
                              {test.test_id}
                            </TableCell>
                            <TableCell className="font-medium">{test.title}</TableCell>
                            <TableCell className="text-muted-foreground">{test.subject_id}</TableCell>
                            <TableCell className="text-muted-foreground">{test.duration} min</TableCell>
                            <TableCell className="text-right">
                              <AlertDialog>
                                <AlertDialogTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="size-8 text-muted-foreground hover:text-destructive"
                                    aria-label={`Delete ${test.title}`}
                                  >
                                    <Trash2Icon className="size-4" />
                                  </Button>
                                </AlertDialogTrigger>
                                <AlertDialogContent>
                                  <AlertDialogHeader>
                                    <AlertDialogTitle>Delete Test</AlertDialogTitle>
                                    <AlertDialogDescription>
                                      Are you sure you want to delete{" "}
                                      <span className="font-semibold">{test.title}</span>?
                                      This action cannot be undone.
                                    </AlertDialogDescription>
                                  </AlertDialogHeader>
                                  <AlertDialogFooter>
                                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                                    <AlertDialogAction
                                      onClick={() => {
                                        setTests((prev) => prev.filter((t) => t.test_id !== test.test_id));
                                        toast.success(`Test "${test.title}" deleted`);
                                      }}
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
                </div>
              )}
            </TabsContent>

            <TabsContent value="assign" className="flex flex-col gap-6">
              <CreateAssignmentForm
                onCreated={(assignment) => setAssignments((prev) => [assignment, ...prev])}
              />

              {assignments.length > 0 && (
                <div className="flex flex-col gap-3">
                  <div className="flex items-center justify-between">
                    <h3 className="text-base font-semibold">Assignments</h3>
                    <Badge variant="secondary">{assignments.length}</Badge>
                  </div>
                  <div className="rounded-lg border overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-16">ID</TableHead>
                          <TableHead>Student ID</TableHead>
                          <TableHead>Test ID</TableHead>
                          <TableHead className="w-20 text-right">Action</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {assignments.map((assignment) => (
                          <TableRow key={assignment.assignment_id}>
                            <TableCell className="font-mono text-sm text-muted-foreground">
                              {assignment.assignment_id}
                            </TableCell>
                            <TableCell className="text-muted-foreground">{assignment.student_id}</TableCell>
                            <TableCell className="text-muted-foreground">{assignment.test_id}</TableCell>
                            <TableCell className="text-right">
                              <AlertDialog>
                                <AlertDialogTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="size-8 text-muted-foreground hover:text-destructive"
                                    aria-label="Delete assignment"
                                  >
                                    <Trash2Icon className="size-4" />
                                  </Button>
                                </AlertDialogTrigger>
                                <AlertDialogContent>
                                  <AlertDialogHeader>
                                    <AlertDialogTitle>Delete Assignment</AlertDialogTitle>
                                    <AlertDialogDescription>
                                      Are you sure you want to delete this assignment?
                                      This action cannot be undone.
                                    </AlertDialogDescription>
                                  </AlertDialogHeader>
                                  <AlertDialogFooter>
                                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                                    <AlertDialogAction
                                      onClick={() => {
                                        setAssignments((prev) => prev.filter((a) => a.assignment_id !== assignment.assignment_id));
                                        toast.success("Assignment deleted");
                                      }}
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
                </div>
              )}
            </TabsContent>
          </Tabs>

        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
