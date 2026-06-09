import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createStudent } from "@/services/coach.service";

export type CoachStudent = {
  student_id: number;
  name: string;
  student_code: string;
  coach_id: number;
};

type Props = {
  onCreated: (student: CoachStudent) => void;
};

export function CreateStudentForm({ onCreated }: Props) {
  const [loading, setLoading] = useState(false);

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);

    const data = {
      name: fd.get("name") as string,
      student_code: fd.get("student_code") as string,
      coach_id: 0,
    };

    try {
      setLoading(true);
      const res = await createStudent(data);
      const newStudent: CoachStudent = {
        student_id: res.student_id,
        name: data.name,
        student_code: data.student_code,
        coach_id: 0,
      };
      onCreated(newStudent);
      toast.success(`Student "${data.name}" created`);
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
        <CardDescription>
          Add a new student to your roster.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="grid gap-4 sm:grid-cols-2">
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
          </div>
          <Button type="submit" disabled={loading} className="w-fit">
            {loading ? "Creating…" : "Create Student"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
