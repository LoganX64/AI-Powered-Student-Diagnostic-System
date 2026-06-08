import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createAssignment, type CreateAssignmentPayload } from "@/services/admin.service";

export function CreateAssignmentForm() {
  const [loading, setLoading] = useState(false);

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);

    const studentId = Number(fd.get("student_id"));
    const testId = Number(fd.get("test_id"));
    const coachId = Number(fd.get("coach_id"));

    if (!studentId || !testId || !coachId) {
      toast.error("All IDs must be valid positive numbers");
      return;
    }

    const data: CreateAssignmentPayload = {
      student_id: studentId,
      test_id: testId,
      coach_id: coachId,
    };

    try {
      setLoading(true);
      const res = await createAssignment(data);
      toast.success(`Test assigned — Assignment ID: ${res.assignment_id}`);
      (e.target as HTMLFormElement).reset();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Assign Test to Student</CardTitle>
        <CardDescription>
          Link a test to a student under a specific coach.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div className="flex flex-col gap-2">
              <Label htmlFor="assign-student-id">Student ID</Label>
              <Input
                id="assign-student-id"
                name="student_id"
                type="number"
                min={1}
                placeholder="1"
                required
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="assign-test-id">Test ID</Label>
              <Input
                id="assign-test-id"
                name="test_id"
                type="number"
                min={1}
                placeholder="1"
                required
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="assign-coach-id">Coach ID</Label>
              <Input
                id="assign-coach-id"
                name="coach_id"
                type="number"
                min={1}
                placeholder="1"
                required
              />
            </div>
          </div>
          <Button type="submit" disabled={loading} className="w-fit">
            {loading ? "Assigning…" : "Assign Test"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
