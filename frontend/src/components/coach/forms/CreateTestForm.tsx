import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createTest } from "@/services/coach.service";

export type CoachTest = {
  test_id: number;
  title: string;
  subject_id: number;
  coach_id: number;
  duration: number;
};

type Props = {
  onCreated?: (test: CoachTest) => void;
};

export function CreateTestForm({ onCreated }: Props) {
  const [loading, setLoading] = useState(false);

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);

    const subjectId = Number(fd.get("subject_id"));
    const duration = Number(fd.get("duration"));

    if (isNaN(subjectId) || subjectId < 1) {
      toast.error("Subject ID must be a valid positive number");
      return;
    }
    if (isNaN(duration) || duration < 1) {
      toast.error("Duration must be a positive number of minutes");
      return;
    }

    const data = {
      title: fd.get("title") as string,
      subject_id: subjectId,
      coach_id: 0,
      duration,
    };

    try {
      setLoading(true);
      const res = await createTest(data);
      const newTest: CoachTest = {
        test_id: res.test_id,
        title: data.title,
        subject_id: data.subject_id,
        coach_id: 0,
        duration: data.duration,
      };
      onCreated?.(newTest);
      toast.success(`Test "${data.title}" created — ID: ${res.test_id}`);
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
        <CardDescription>Create a new test under a subject.</CardDescription>
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
          </div>
          <Button type="submit" disabled={loading} className="w-fit">
            {loading ? "Creating…" : "Create Test"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
