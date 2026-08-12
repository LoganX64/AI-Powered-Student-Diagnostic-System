import { useState, useRef, useEffect, useCallback } from "react";
import { toast } from "sonner";
import { createStudentSchema, zodErrors } from "@/lib/validations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createStudent, getCoaches, type Coach } from "@/services/dashboard.service";

export function CreateStudentForm() {
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [coachSearch, setCoachSearch] = useState("");
  const [selectedCoach, setSelectedCoach] = useState<Coach | null>(null);
  const [coaches, setCoaches] = useState<Coach[]>([]);
  const [showDropdown, setShowDropdown] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const fetchCoaches = useCallback(async (search: string) => {
    try {
      const res = await getCoaches({ search, limit: 20 });
      setCoaches(res.data ?? []);
    } catch {
      setCoaches([]);
    }
  }, []);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      fetchCoaches(coachSearch);
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [coachSearch, fetchCoaches]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleSelectCoach = (coach: Coach) => {
    setSelectedCoach(coach);
    setCoachSearch(coach.name);
    setShowDropdown(false);
  };

  const handleCoachInputChange = (value: string) => {
    setCoachSearch(value);
    setSelectedCoach(null);
    setShowDropdown(true);
  };

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();

    const fd = new FormData(e.currentTarget);
    const raw = {
      name: fd.get("name") as string,
      student_code: fd.get("student_code") as string,
      coach_id: selectedCoach?.coach_id ?? 0,
    };

    const result = createStudentSchema.safeParse(raw);
    if (!result.success) {
      setErrors(zodErrors(result.error));
      return;
    }
    setErrors({});

    try {
      setLoading(true);
      const res = await createStudent(result.data);
      toast.success(`Student created — ID: ${res.student_id}`);
      (e.target as HTMLFormElement).reset();
      setCoachSearch("");
      setSelectedCoach(null);
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
            {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="student-code">Student Code</Label>
            <Input
              id="student-code"
              name="student_code"
              placeholder="STU001"
              required
            />
            {errors.student_code && <p className="text-sm text-destructive">{errors.student_code}</p>}
          </div>
          <div className="flex flex-col gap-2 relative" ref={dropdownRef}>
            <Label htmlFor="student-coach">Coach</Label>
            <Input
              id="student-coach"
              placeholder="Search coach by name…"
              value={coachSearch}
              onChange={(e) => handleCoachInputChange(e.target.value)}
              onFocus={() => setShowDropdown(true)}
              required
            />
            {showDropdown && coaches.length > 0 && (
              <div className="absolute top-full left-0 right-0 z-50 mt-1 max-h-60 overflow-auto rounded-md border bg-popover text-popover-foreground shadow-md">
                {coaches.map((coach) => (
                  <button
                    type="button"
                    key={coach.coach_id}
                    className={`flex w-full items-center px-3 py-2 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground ${
                      selectedCoach?.coach_id === coach.coach_id ? "bg-accent text-accent-foreground" : ""
                    }`}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => handleSelectCoach(coach)}
                  >
                    {coach.name}
                  </button>
                ))}
              </div>
            )}
            {showDropdown && coachSearch && coaches.length === 0 && (
              <div className="absolute top-full left-0 right-0 z-50 mt-1 rounded-md border bg-popover px-3 py-2 text-sm text-muted-foreground shadow-md">
                No coaches found
              </div>
            )}
            {errors.coach_id && <p className="text-sm text-destructive">{errors.coach_id}</p>}
          </div>
          <Button type="submit" disabled={loading} className="w-fit">
            {loading ? "Creating…" : "Create Student"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
