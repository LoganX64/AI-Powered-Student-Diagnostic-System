import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { ClipboardListIcon, UsersIcon } from "lucide-react";
import { useRole } from "@/hooks/useRole";
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
import { createAssignment as adminCreateAssignment, getStudents as adminGetStudents, getTests as adminGetTests, type Student, type Test } from "@/services/admin.service";

type Props = {
  onSubmit?: (data: { student_id: number; test_id: number; coach_id: number }) => Promise<{ assignment_id: number }>;
  fetchStudents?: (params?: { limit?: number }) => Promise<{ data: Student[] }>;
  fetchTests?: (params?: { limit?: number }) => Promise<{ data: Test[] }>;
};

export function CreateAssignmentForm({ onSubmit, fetchStudents, fetchTests }: Props) {
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";
  const [loading, setLoading] = useState(false);
  const [students, setStudents] = useState<Student[]>([]);
  const [tests, setTests] = useState<Test[]>([]);
  const [selectedStudentId, setSelectedStudentId] = useState("");
  const [selectedTestId, setSelectedTestId] = useState("");

  useEffect(() => {
    const getStudentsFn = fetchStudents ?? adminGetStudents;
    const getTestsFn = fetchTests ?? adminGetTests;
    getStudentsFn({ limit: 10000 }).then((res) => setStudents(res.data ?? [])).catch(() => {});
    getTestsFn({ limit: 10000 }).then((res) => setTests(res.data ?? [])).catch(() => {});
  }, [fetchStudents, fetchTests]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const studentId = Number(selectedStudentId);
    const testId = Number(selectedTestId);

    if (!studentId || !testId) {
      toast.error("Please select both a student and a test");
      return;
    }

    const data = {
      student_id: studentId,
      test_id: testId,
      coach_id: tests.find((t) => t.test_id === testId)?.coach_id ?? 0,
    };

    try {
      setLoading(true);
      const createFn = onSubmit ?? adminCreateAssignment;
      const res = await createFn(data);
      toast.success(`Test assigned — Assignment ID: ${res.assignment_id}`);
      setSelectedStudentId("");
      setSelectedTestId("");
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  if (tests.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center gap-3 py-12">
          <ClipboardListIcon className="size-8 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No tests created yet.</p>
          <Button variant="outline" size="sm" onClick={() => navigate(`${prefix}/tests`)}>
            Create a Test
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (students.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center gap-3 py-12">
          <UsersIcon className="size-8 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No students added yet.</p>
          <Button variant="outline" size="sm" onClick={() => navigate(`${prefix}/students`)}>
            Add a Student
          </Button>
        </CardContent>
      </Card>
    );
  }

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
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-2">
              <Label>Student</Label>
              <Select value={selectedStudentId} onValueChange={setSelectedStudentId}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a student" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {students.map((s) => (
                      <SelectItem key={s.student_id} value={s.student_id.toString()}>
                        {s.name} ({s.student_code})
                      </SelectItem>
                    ))}
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
                    {tests.map((t) => (
                      <SelectItem key={t.test_id} value={t.test_id.toString()}>
                        {t.title} (ID: {t.test_id})
                      </SelectItem>
                    ))}
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
