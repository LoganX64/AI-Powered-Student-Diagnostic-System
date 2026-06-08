import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createStudent, type CreateStudentPayload } from "@/services/admin.service";

export function CreateStudentForm() {
  const [loading, setLoading] = useState(false);

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const coachIdRaw = fd.get("coach_id") as string;

    if (!coachIdRaw || isNaN(Number(coachIdRaw))) {
      toast.error("Coach ID must be a valid number");
      return;
    }

    const data: CreateStudentPayload = {
      name: fd.get("name") as string,
      student_code: fd.get("student_code") as string,
      coach_id: Number(coachIdRaw),
    };

    try {
      setLoading(true);
      const res = await createStudent(data);
      toast.success(`Student created — ID: ${res.student_id}`);
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
        <CardTitle>Create Student</CardTitle>
        <CardDescription>Add a new student and assign them to a coach.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="student-name">Full Name</Label>
            <Input
              id="student-name"
              name="name"
              placeholder="Alice"
              required
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="student-code">Student Code</Label>
            <Input
              id="student-code"
              name="student_code"
              placeholder="STU001"
              required
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="student-coach-id">Coach ID</Label>
            <Input
              id="student-coach-id"
              name="coach_id"
              type="number"
              min={1}
              placeholder="1"
              required
            />
          </div>
          <Button type="submit" disabled={loading} className="w-fit">
            {loading ? "Creating…" : "Create Student"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
