import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CreateTestForm } from "@/components/admin/forms/CreateTestForm";
import { CreateQuestionsForm } from "@/components/admin/forms/CreateQuestionsForm";
import { CreateAssignmentForm } from "@/components/admin/forms/CreateAssignmentForm";
import { ClipboardListIcon, LinkIcon } from "lucide-react";

const TABS = [
  { value: "test", label: "Create Test & Questions", icon: ClipboardListIcon },
  { value: "assign", label: "Assign", icon: LinkIcon },
];

export function TestsPage() {
  const [createdTestId, setCreatedTestId] = useState<number | null>(null);

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

            <TabsContent value="test" className="flex flex-col gap-6">
              {createdTestId === null ? (
                <CreateTestForm onCreated={(id) => setCreatedTestId(id)} />
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
