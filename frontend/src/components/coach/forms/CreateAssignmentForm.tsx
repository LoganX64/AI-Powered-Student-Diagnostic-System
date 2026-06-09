import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createAssignment } from "@/services/coach.service";

export type CoachAssignment = {
  assignment_id: number;
  student_id: number;
  test_id: number;
  coach_id: number;
};

type Props = {
  onCreated: (assignment: CoachAssignment) => void;
};

export function CreateAssignmentForm({ onCreated }: Props) {
  const [loading, setLoading] = useState(false);

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);

    const studentId = Number(fd.get("student_id"));
    const testId = Number(fd.get("test_id"));

    if (!studentId || !testId) {
      toast.error("All IDs must be valid positive numbers");
      return;
    }

    try {
      setLoading(true);
      const res = await createAssignment({
        student_id: studentId,
        test_id: testId,
        coach_id: 0,
      });
      const newAssignment: CoachAssignment = {
        assignment_id: res.assignment_id,
        student_id: studentId,
        test_id: testId,
        coach_id: 0,
      };
      onCreated(newAssignment);
      toast.success(`Test assigned to student ${studentId}`);
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
          Link a test to a student.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
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
          </div>
          <Button type="submit" disabled={loading} className="w-fit">
            {loading ? "Assigning…" : "Assign Test"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
