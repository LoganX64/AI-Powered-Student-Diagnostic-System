import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CreateCoachForm } from "@/components/admin/forms/CreateCoachForm";
import { CreateStudentForm } from "@/components/admin/forms/CreateStudentForm";
import { CreateSubjectForm } from "@/components/admin/forms/CreateSubjectForm";
import { CreateTestForm } from "@/components/admin/forms/CreateTestForm";
import { CreateQuestionsForm } from "@/components/admin/forms/CreateQuestionsForm";
import { CreateAssignmentForm } from "@/components/admin/forms/CreateAssignmentForm";

const TABS = [
  { value: "coach", label: "Coach" },
  { value: "student", label: "Student" },
  { value: "subject", label: "Subject" },
  { value: "test", label: "Test" },
  { value: "questions", label: "Questions" },
  { value: "assign", label: "Assign" },
];

export function AdminManagePage() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <SiteHeader title="Manage" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">
          <div>
            <h2 className="text-lg font-semibold">Create &amp; Manage</h2>
            <p className="text-sm text-muted-foreground">
              Use the tabs below to create coaches, students, subjects, tests,
              questions, and assignments.
            </p>
          </div>

          <Tabs defaultValue="coach" className="w-full">
            <TabsList className="mb-4 flex flex-wrap gap-1 h-auto">
              {TABS.map((t) => (
                <TabsTrigger key={t.value} value={t.value}>
                  {t.label}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent value="coach">
              <CreateCoachForm />
            </TabsContent>
            <TabsContent value="student">
              <CreateStudentForm />
            </TabsContent>
            <TabsContent value="subject">
              <CreateSubjectForm />
            </TabsContent>
            <TabsContent value="test">
              <CreateTestForm />
            </TabsContent>
            <TabsContent value="questions">
              <CreateQuestionsForm />
            </TabsContent>
            <TabsContent value="assign">
              <CreateAssignmentForm />
            </TabsContent>
          </Tabs>
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
