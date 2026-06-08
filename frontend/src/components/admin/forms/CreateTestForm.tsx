import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createTest, type CreateTestPayload } from "@/services/admin.service";

export function CreateTestForm() {
  const [loading, setLoading] = useState(false);

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);

    const subjectId = Number(fd.get("subject_id"));
    const coachId = Number(fd.get("coach_id"));
    const duration = Number(fd.get("duration"));

    if (isNaN(subjectId) || subjectId < 1) {
      toast.error("Subject ID must be a valid positive number");
      return;
    }
    if (isNaN(coachId) || coachId < 1) {
      toast.error("Coach ID must be a valid positive number");
      return;
    }
    if (isNaN(duration) || duration < 1) {
      toast.error("Duration must be a positive number of minutes");
      return;
    }

    const data: CreateTestPayload = {
      title: fd.get("title") as string,
      subject_id: subjectId,
      coach_id: coachId,
      duration,
    };

    try {
      setLoading(true);
      const res = await createTest(data);
      toast.success(`Test created — ID: ${res.test_id}`);
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
        <CardTitle>Create Test</CardTitle>
        <CardDescription>Create a new test under a subject and assign it to a coach.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="test-title">Title</Label>
            <Input
              id="test-title"
              name="title"
              placeholder="Mathematics Midterm Exam 2026"
              required
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="test-subject-id">Subject ID</Label>
              <Input
                id="test-subject-id"
                name="subject_id"
                type="number"
                min={1}
                placeholder="1"
                required
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="test-coach-id">Coach ID</Label>
              <Input
                id="test-coach-id"
                name="coach_id"
                type="number"
                min={1}
                placeholder="1"
                required
              />
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="test-duration">Duration (minutes)</Label>
            <Input
              id="test-duration"
              name="duration"
              type="number"
              min={1}
              placeholder="120"
              required
            />
          </div>
          <Button type="submit" disabled={loading} className="w-fit">
            {loading ? "Creating…" : "Create Test"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
