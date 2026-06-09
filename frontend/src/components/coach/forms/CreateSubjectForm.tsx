import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createSubject } from "@/services/coach.service";

export type CoachSubject = {
  subject_id: number;
  name: string;
};

type Props = {
  onCreated: (subject: CoachSubject) => void;
};

export function CreateSubjectForm({ onCreated }: Props) {
  const [loading, setLoading] = useState(false);

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const name = fd.get("name") as string;

    try {
      setLoading(true);
      const res = await createSubject({ name });
      const newSubject: CoachSubject = {
        subject_id: res.subject_id,
        name,
      };
      onCreated(newSubject);
      toast.success(`Subject "${name}" created`);
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
        <CardTitle>Create Subject</CardTitle>
        <CardDescription>Add a new subject to your curriculum.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="flex gap-3 items-end">
            <div className="flex flex-col gap-2 max-w-sm w-full">
              <Label htmlFor="subject-name">Subject Name</Label>
              <Input
                id="subject-name"
                name="name"
                placeholder="Mathematics"
                required
              />
            </div>
            <Button type="submit" disabled={loading}>
              {loading ? "Creating…" : "Create Subject"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
