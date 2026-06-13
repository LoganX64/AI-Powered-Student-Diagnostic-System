import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { CreateTestForm } from "@/components/admin/forms/CreateTestForm";
import { CreateQuestionsForm } from "@/components/admin/forms/CreateQuestionsForm";
import { CreateAssignmentForm } from "@/components/admin/forms/CreateAssignmentForm";
import { deleteTest as adminDeleteTest } from "@/services/dashboard.service";
import {
  createTest as coachCreateTest,
  createQuestions as coachCreateQuestions,
  createAssignment as coachCreateAssignment,
  deleteTest as coachDeleteTest,
  getStudents as coachGetStudents,
  getTests as coachGetTests,
  getSubjects as coachGetSubjects,
} from "@/services/coach.service";
import { ClipboardListIcon, LinkIcon } from "lucide-react";

const TABS = [
  { value: "test", label: "Create Test & Questions", icon: ClipboardListIcon },
  { value: "assign", label: "Assign", icon: LinkIcon },
];

export function TestsPage() {
  const navigate = useNavigate();
  const role = useRole();
  const isCoach = role === "coach";
  const prefix = isCoach ? "/coach" : "/admin";
  const [createdTestId, setCreatedTestId] = useState<number | null>(null);
  const [showExitDialog, setShowExitDialog] = useState(false);

  const handleDeleteTest = async () => {
    if (createdTestId === null) return;
    try {
      const deleteFn = isCoach ? coachDeleteTest : adminDeleteTest;
      await deleteFn(createdTestId);
      toast.success("Empty test deleted");
    } catch {
      // silently ignore — test may already be gone
    } finally {
      setCreatedTestId(null);
      setShowExitDialog(false);
    }
  };

  return (
    <DashboardLayout title="Tests">
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
              onCreated={(id) => setCreatedTestId(id)}
              onSubmit={isCoach ? coachCreateTest : undefined}
              showCoachField={!isCoach}
              fetchSubjects={isCoach ? coachGetSubjects : undefined}
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
                  onClick={() => setShowExitDialog(true)}
                >
                  Create Another Test
                </Button>
              </div>
              <CreateQuestionsForm
                testId={createdTestId}
                onCreated={(id, count) => {
                  toast.success(`${count} question(s) added to test ${id}`);
                  setCreatedTestId(null);
                  navigate(`${prefix}/all-tests`);
                }}
                onSubmit={isCoach ? coachCreateQuestions : undefined}
              />
            </>
          )}
        </TabsContent>

        <TabsContent value="assign">
          <CreateAssignmentForm
            onSubmit={isCoach ? coachCreateAssignment : undefined}
            fetchStudents={isCoach ? coachGetStudents : undefined}
            fetchTests={isCoach ? coachGetTests : undefined}
          />
        </TabsContent>
      </Tabs>

      <AlertDialog open={showExitDialog} onOpenChange={setShowExitDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Test has no questions</AlertDialogTitle>
            <AlertDialogDescription>
              This test has no questions yet. Do you want to add questions or delete this empty test?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteTest}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete Test
            </AlertDialogAction>
            <AlertDialogAction onClick={() => setShowExitDialog(false)}>
              Add Questions
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </DashboardLayout>
  );
}
