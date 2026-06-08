import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CreateTestForm } from "@/components/admin/forms/CreateTestForm";
import { CreateQuestionsForm } from "@/components/admin/forms/CreateQuestionsForm";
import { CreateAssignmentForm } from "@/components/admin/forms/CreateAssignmentForm";
import { ClipboardListIcon, HelpCircleIcon, LinkIcon } from "lucide-react";

const TABS = [
  { value: "test", label: "Test", icon: ClipboardListIcon },
  { value: "questions", label: "Questions", icon: HelpCircleIcon },
  { value: "assign", label: "Assign", icon: LinkIcon },
];

export function TestsPage() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <SiteHeader title="Tests" />
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
