import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { CreateAssignmentForm } from "@/components/admin/forms/CreateAssignmentForm";

export function AssignTestPage() {
  return (
    <DashboardLayout title="Assign Test">
      <div>
        <h2 className="text-lg font-semibold">Assign Test</h2>
        <p className="text-sm text-muted-foreground">
          Select a test and assign it to a student or batch.
        </p>
      </div>
      <CreateAssignmentForm />
    </DashboardLayout>
  );
}
