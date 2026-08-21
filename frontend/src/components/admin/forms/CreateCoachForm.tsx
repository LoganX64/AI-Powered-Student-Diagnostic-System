import { useState } from "react";
import { toast } from "sonner";
import { createCoachSchema, zodErrors } from "@/lib/validations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createCoach } from "@/services/dashboard.service";
import { SubjectPicker } from "./SubjectPicker";

export function CreateCoachForm() {
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [selectedSubjectIds, setSelectedSubjectIds] = useState<number[]>([]);

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const raw = {
      email: fd.get("email") as string,
      password: fd.get("password") as string,
      name: fd.get("name") as string,
      subject_ids: selectedSubjectIds,
    };

    const result = createCoachSchema.safeParse(raw);
    if (!result.success) {
      setErrors(zodErrors(result.error));
      return;
    }
    setErrors({});

    try {
      setLoading(true);
      const res = await createCoach(result.data);
      toast.success(`Coach created — ID: ${res.coach_id}`);
      (e.target as HTMLFormElement).reset();
      setSelectedSubjectIds([]);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Create Coach</CardTitle>
        <CardDescription>Add a new coach to your organization.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="coach-name">Full Name</Label>
            <Input
              id="coach-name"
              name="name"
              placeholder="John Smith"
              required
            />
            {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="coach-email">Email</Label>
            <Input
              id="coach-email"
              name="email"
              type="email"
              placeholder="coach.smith@academy.com"
              required
            />
            {errors.email && <p className="text-sm text-destructive">{errors.email}</p>}
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="coach-password">Password</Label>
            <Input
              id="coach-password"
              name="password"
              type="password"
              placeholder="Min. 8 characters"
              required
              minLength={8}
            />
            {errors.password && <p className="text-sm text-destructive">{errors.password}</p>}
          </div>
          <div className="flex flex-col gap-2">
            <Label>Subjects *</Label>
            <SubjectPicker
              selected={selectedSubjectIds}
              onChange={setSelectedSubjectIds}
              error={errors.subject_ids}
            />
          </div>
          <Button type="submit" disabled={loading} className="w-fit">
            {loading ? "Creating…" : "Create Coach"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
