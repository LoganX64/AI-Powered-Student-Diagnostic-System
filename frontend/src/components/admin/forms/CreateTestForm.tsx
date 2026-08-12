import { useState, useRef, useEffect, useCallback } from "react";
import { toast } from "sonner";
import { createTestSchema, zodErrors } from "@/lib/validations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { createTest as adminCreateTest, getCoaches as adminGetCoaches, getSubjects as adminGetSubjects, type CreateTestPayload, type Coach, type Subject, type PaginationParams, type PaginatedResponse } from "@/services/dashboard.service";

type Props = {
  onCreated?: (testId: number) => void;
  onSubmit?: (data: CreateTestPayload) => Promise<{ test_id: number }>;
  showCoachField?: boolean;
  fetchSubjects?: (params?: PaginationParams) => Promise<PaginatedResponse<Subject>>;
  fetchCoaches?: (params?: PaginationParams) => Promise<PaginatedResponse<Coach>>;
};

export function CreateTestForm({ onCreated, onSubmit, showCoachField = true, fetchSubjects: fetchSubjectsProp, fetchCoaches: fetchCoachesProp }: Props) {
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const [subjectSearch, setSubjectSearch] = useState("");
  const [selectedSubject, setSelectedSubject] = useState<Subject | null>(null);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [showSubjectDropdown, setShowSubjectDropdown] = useState(false);
  const subjectInputRef = useRef<HTMLInputElement>(null);
  const subjectDropdownRef = useRef<HTMLDivElement>(null);
  const [subjectDropdownPos, setSubjectDropdownPos] = useState({ top: 0, left: 0, width: 0 });

  const [coachSearch, setCoachSearch] = useState("");
  const [selectedCoach, setSelectedCoach] = useState<Coach | null>(null);
  const [coaches, setCoaches] = useState<Coach[]>([]);
  const [showCoachDropdown, setShowCoachDropdown] = useState(false);
  const coachInputRef = useRef<HTMLInputElement>(null);
  const coachDropdownRef = useRef<HTMLDivElement>(null);
  const [coachDropdownPos, setCoachDropdownPos] = useState({ top: 0, left: 0, width: 0 });

  const subjectDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const coachDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const fetchSubjects = useCallback(async (search: string) => {
    try {
      const fn = fetchSubjectsProp ?? adminGetSubjects;
      const res = await fn({ search, limit: 10 });
      setSubjects(res.data ?? []);
    } catch {
      setSubjects([]);
    }
  }, [fetchSubjectsProp]);

  const fetchCoaches = useCallback(async (search: string) => {
    try {
      const fn = fetchCoachesProp ?? adminGetCoaches;
      const res = await fn({ search, limit: 10 });
      setCoaches(res.data ?? []);
    } catch {
      setCoaches([]);
    }
  }, [fetchCoachesProp]);

  useEffect(() => {
    if (subjectDebounceRef.current) clearTimeout(subjectDebounceRef.current);
    subjectDebounceRef.current = setTimeout(() => fetchSubjects(subjectSearch), 300);
    return () => { if (subjectDebounceRef.current) clearTimeout(subjectDebounceRef.current); };
  }, [subjectSearch, fetchSubjects]);

  useEffect(() => {
    if (!showCoachField) return;
    if (coachDebounceRef.current) clearTimeout(coachDebounceRef.current);
    coachDebounceRef.current = setTimeout(() => fetchCoaches(coachSearch), 300);
    return () => { if (coachDebounceRef.current) clearTimeout(coachDebounceRef.current); };
  }, [coachSearch, fetchCoaches, showCoachField]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (subjectDropdownRef.current && !subjectDropdownRef.current.contains(e.target as Node)) {
        setShowSubjectDropdown(false);
      }
      if (coachDropdownRef.current && !coachDropdownRef.current.contains(e.target as Node)) {
        setShowCoachDropdown(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleSelectSubject = (subject: Subject) => {
    setSelectedSubject(subject);
    setSubjectSearch(subject.name);
    setShowSubjectDropdown(false);
  };

  const handleSubjectInputChange = (value: string) => {
    setSubjectSearch(value);
    setSelectedSubject(null);
    setShowSubjectDropdown(true);
    if (subjectInputRef.current) {
      const rect = subjectInputRef.current.getBoundingClientRect();
      setSubjectDropdownPos({ top: rect.bottom + 4, left: rect.left, width: rect.width });
    }
  };

  const handleSelectCoach = (coach: Coach) => {
    setSelectedCoach(coach);
    setCoachSearch(coach.name);
    setShowCoachDropdown(false);
  };

  const handleCoachInputChange = (value: string) => {
    setCoachSearch(value);
    setSelectedCoach(null);
    setShowCoachDropdown(true);
    if (coachInputRef.current) {
      const rect = coachInputRef.current.getBoundingClientRect();
      setCoachDropdownPos({ top: rect.bottom + 4, left: rect.left, width: rect.width });
    }
  };

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();

    const fd = new FormData(e.currentTarget);
    const raw = {
      title: fd.get("title") as string,
      subject_id: selectedSubject?.subject_id ?? 0,
      subject_name: selectedSubject?.name ?? "",
      coach_id: selectedCoach?.coach_id ?? 0,
      duration: Number(fd.get("duration")),
      exam_date: (fd.get("exam_date") as string) || undefined,
    };

    const result = createTestSchema.safeParse(raw);
    if (!result.success) {
      setErrors(zodErrors(result.error));
      return;
    }
    setErrors({});

    try {
      setLoading(true);
      const createFn = onSubmit ?? adminCreateTest;
      const res = await createFn(result.data);
      toast.success(`Test created — ID: ${res.test_id}`);
      onCreated?.(res.test_id);
      (e.target as HTMLFormElement).reset();
      setSubjectSearch("");
      setSelectedSubject(null);
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
            {errors.title && <p className="text-sm text-destructive">{errors.title}</p>}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="test-subject">Subject</Label>
              <Input
                ref={subjectInputRef}
                id="test-subject"
                placeholder="Search subject by name…"
                value={subjectSearch}
                onChange={(e) => handleSubjectInputChange(e.target.value)}
                onFocus={() => {
                  setShowSubjectDropdown(true);
                  if (subjectInputRef.current) {
                    const rect = subjectInputRef.current.getBoundingClientRect();
                    setSubjectDropdownPos({ top: rect.bottom + 4, left: rect.left, width: rect.width });
                  }
                }}
                required
              />
              {showSubjectDropdown && subjects.length > 0 && (
                <div
                  ref={subjectDropdownRef}
                  style={{ position: "fixed", top: subjectDropdownPos.top, left: subjectDropdownPos.left, width: subjectDropdownPos.width }}
                  className="z-50 max-h-60 overflow-auto rounded-md border bg-popover text-popover-foreground shadow-md"
                >
                  {subjects.map((subject) => (
                    <button
                      type="button"
                      key={subject.subject_id}
                      className={`flex w-full items-center px-3 py-2 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground ${
                        selectedSubject?.subject_id === subject.subject_id ? "bg-accent text-accent-foreground" : ""
                      }`}
                      onMouseDown={(e) => e.preventDefault()}
                      onClick={() => handleSelectSubject(subject)}
                    >
                      {subject.name}
                    </button>
                  ))}
                </div>
              )}
              {showSubjectDropdown && subjectSearch && subjects.length === 0 && (
                <div
                  style={{ position: "fixed", top: subjectDropdownPos.top, left: subjectDropdownPos.left, width: subjectDropdownPos.width }}
                  className="z-50 rounded-md border bg-popover px-3 py-2 text-sm text-muted-foreground shadow-md"
                >
                  No subjects found
                </div>
              )}
              {errors.subject_id && <p className="text-sm text-destructive">{errors.subject_id}</p>}
            </div>
            {showCoachField && (
            <div className="flex flex-col gap-2">
              <Label htmlFor="test-coach">Coach</Label>
              <Input
                ref={coachInputRef}
                id="test-coach"
                placeholder="Search coach by name…"
                value={coachSearch}
                onChange={(e) => handleCoachInputChange(e.target.value)}
                onFocus={() => {
                  setShowCoachDropdown(true);
                  if (coachInputRef.current) {
                    const rect = coachInputRef.current.getBoundingClientRect();
                    setCoachDropdownPos({ top: rect.bottom + 4, left: rect.left, width: rect.width });
                  }
                }}
                required
              />
              {showCoachDropdown && coaches.length > 0 && (
                <div
                  ref={coachDropdownRef}
                  style={{ position: "fixed", top: coachDropdownPos.top, left: coachDropdownPos.left, width: coachDropdownPos.width }}
                  className="z-50 max-h-60 overflow-auto rounded-md border bg-popover text-popover-foreground shadow-md"
                >
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
              {showCoachDropdown && coachSearch && coaches.length === 0 && (
                <div
                  style={{ position: "fixed", top: coachDropdownPos.top, left: coachDropdownPos.left, width: coachDropdownPos.width }}
                  className="z-50 rounded-md border bg-popover px-3 py-2 text-sm text-muted-foreground shadow-md"
                >
                  No coaches found
                </div>
              )}
              {errors.coach_id && <p className="text-sm text-destructive">{errors.coach_id}</p>}
            </div>
            )}
          </div>
          <div className="grid grid-cols-2 gap-4">
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
              {errors.duration && <p className="text-sm text-destructive">{errors.duration}</p>}
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="test-exam-date">Exam Date</Label>
              <Input
                id="test-exam-date"
                name="exam_date"
                type="date"
              />
              {errors.exam_date && <p className="text-sm text-destructive">{errors.exam_date}</p>}
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
