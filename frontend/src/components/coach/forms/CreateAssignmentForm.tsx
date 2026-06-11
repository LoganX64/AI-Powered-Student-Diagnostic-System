import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { createAssignment, getStudents, getTests } from "@/services/coach.service";
import type { Student, Test } from "@/services/admin.service";

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
  const [students, setStudents] = useState<Student[]>([]);
  const [tests, setTests] = useState<Test[]>([]);
  const [selectedStudentId, setSelectedStudentId] = useState("");
  const [selectedTestId, setSelectedTestId] = useState("");

  useEffect(() => {
    getStudents({ limit: 10000 }).then((res) => setStudents(res.data)).catch(() => {});
    getTests({ limit: 10000 }).then((res) => setTests(res.data)).catch(() => {});
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const studentId = Number(selectedStudentId);
    const testId = Number(selectedTestId);

    if (!studentId || !testId) {
      toast.error("Please select both a student and a test");
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
      setSelectedStudentId("");
      setSelectedTestId("");
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
              <Label>Student</Label>
              <Select value={selectedStudentId} onValueChange={setSelectedStudentId}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a student" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {students.length === 0 ? (
                      <SelectItem value="none" disabled>No students found</SelectItem>
                    ) : (
                      students.map((s) => (
                        <SelectItem key={s.student_id} value={s.student_id.toString()}>
                          {s.name} ({s.student_code})
                        </SelectItem>
                      ))
                    )}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-2">
              <Label>Test</Label>
              <Select value={selectedTestId} onValueChange={setSelectedTestId}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a test" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {tests.length === 0 ? (
                      <SelectItem value="none" disabled>No tests found</SelectItem>
                    ) : (
                      tests.map((t) => (
                        <SelectItem key={t.test_id} value={t.test_id.toString()}>
                          {t.title} (ID: {t.test_id})
                        </SelectItem>
                      ))
                    )}
                  </SelectGroup>
                </SelectContent>
              </Select>
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
