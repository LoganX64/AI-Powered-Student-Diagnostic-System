import { useState } from "react";
import { toast } from "sonner";
import { createSubjectSchema, zodErrors } from "@/lib/validations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createSubject } from "@/services/dashboard.service";

export function CreateSubjectForm() {
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const raw = { name: fd.get("name") as string };

    const result = createSubjectSchema.safeParse(raw);
    if (!result.success) {
      setErrors(zodErrors(result.error));
      return;
    }
    setErrors({});

    try {
      setLoading(true);
      const res = await createSubject(result.data);
      toast.success(`Subject created — ID: ${res.subject_id}`);
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
        <CardDescription>Add a new subject to your organization.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="subject-name">Subject Name</Label>
            <Input
              id="subject-name"
              name="name"
              placeholder="Mathematics"
              required
            />
            {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
          </div>
          <Button type="submit" disabled={loading} className="w-fit">
            {loading ? "Creating…" : "Create Subject"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
