import { useState, useEffect } from "react";
import { toast } from "sonner";
import { updateCoachSchema, zodErrors } from "@/lib/validations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { updateCoach, type Coach } from "@/services/dashboard.service";
import { SubjectPicker } from "@/components/admin/forms/SubjectPicker";

type Props = {
  coach: Coach | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated?: () => void;
};

export function EditCoachDialog({ coach, open, onOpenChange, onUpdated }: Props) {
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [selectedSubjectIds, setSelectedSubjectIds] = useState<number[]>([]);

  useEffect(() => {
    if (coach && open) {
      setName(coach.name);
      setEmail(coach.email);
      setSelectedSubjectIds(coach.subjects?.map((s) => s.subject_id) ?? []);
      setErrors({});
    }
  }, [coach, open]);

  const handleSave = async () => {
    if (!coach) return;

    const raw = {
      name: name.trim(),
      email: email.trim(),
      subject_ids: selectedSubjectIds,
    };

    const result = updateCoachSchema.safeParse(raw);
    if (!result.success) {
      setErrors(zodErrors(result.error));
      return;
    }
    setErrors({});

    try {
      setLoading(true);
      await updateCoach(coach.coach_id, result.data);
      toast.success("Coach updated");
      onOpenChange(false);
      onUpdated?.();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  if (!coach) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            Edit Coach
            <Badge variant="outline">ID: {coach.coach_id}</Badge>
          </DialogTitle>
          <DialogDescription>Update coach details below.</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-coach-name">Full Name</Label>
            <Input
              id="edit-coach-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="John Smith"
            />
            {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-coach-email">Email</Label>
            <Input
              id="edit-coach-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="coach@academy.com"
            />
            {errors.email && <p className="text-sm text-destructive">{errors.email}</p>}
          </div>

          <div className="flex flex-col gap-2">
            <Label>Subjects</Label>
            <SubjectPicker
              selected={selectedSubjectIds}
              onChange={setSelectedSubjectIds}
              error={errors.subject_ids}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={handleSave} disabled={loading}>
            {loading ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
