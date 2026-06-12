import { useState, useRef, useEffect, useCallback } from "react";
import { toast } from "sonner";
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
import { getCoaches, getSubjects, updateTest, type Test, type Coach, type Subject } from "@/services/admin.service";

type Props = {
  test: Test | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated?: () => void;
};

export function EditTestDialog({ test, open, onOpenChange, onUpdated }: Props) {
  const [loading, setLoading] = useState(false);

  const [title, setTitle] = useState("");
  const [duration, setDuration] = useState(0);
  const [examDate, setExamDate] = useState("");

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

  const subjectDebounceRef = useRef<ReturnType<typeof setTimeout>>();
  const coachDebounceRef = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    if (test && open) {
      setTitle(test.title);
      setDuration(test.duration);
      setExamDate(test.exam_date ? test.exam_date.substring(0, 10) : "");
      setSubjectSearch(test.subject_name || "");
      setSelectedSubject({ subject_id: test.subject_id, name: test.subject_name || "" });
      setCoachSearch(test.coach_name || "");
      setSelectedCoach({ coach_id: test.coach_id, user_id: 0, name: test.coach_name || "", email: "" });
    }
  }, [test, open]);

  const fetchSubjects = useCallback(async (search: string) => {
    try {
      const res = await getSubjects({ search, limit: 10 });
      setSubjects(res.data ?? []);
    } catch {
      setSubjects([]);
    }
  }, []);

  const fetchCoaches = useCallback(async (search: string) => {
    try {
      const res = await getCoaches({ search, limit: 10 });
      setCoaches(res.data ?? []);
    } catch {
      setCoaches([]);
    }
  }, []);

  useEffect(() => {
    if (subjectDebounceRef.current) clearTimeout(subjectDebounceRef.current);
    subjectDebounceRef.current = setTimeout(() => fetchSubjects(subjectSearch), 300);
    return () => { if (subjectDebounceRef.current) clearTimeout(subjectDebounceRef.current); };
  }, [subjectSearch, fetchSubjects]);

  useEffect(() => {
    if (coachDebounceRef.current) clearTimeout(coachDebounceRef.current);
    coachDebounceRef.current = setTimeout(() => fetchCoaches(coachSearch), 300);
    return () => { if (coachDebounceRef.current) clearTimeout(coachDebounceRef.current); };
  }, [coachSearch, fetchCoaches]);

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

  const handleSave = async () => {
    if (!test) return;
    if (!title.trim()) {
      toast.error("Title is required");
      return;
    }
    if (!selectedSubject) {
      toast.error("Please select a subject");
      return;
    }
    if (!selectedCoach) {
      toast.error("Please select a coach");
      return;
    }
    if (isNaN(duration) || duration < 1) {
      toast.error("Duration must be a positive number");
      return;
    }

    try {
      setLoading(true);
      await updateTest(test.test_id, {
        title: title.trim(),
        subject_id: selectedSubject.subject_id,
        coach_id: selectedCoach.coach_id,
        duration,
        exam_date: examDate || undefined,
      });
      toast.success("Test updated");
      onOpenChange(false);
      onUpdated?.();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  if (!test) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            Edit Test
            <Badge variant="outline">ID: {test.test_id}</Badge>
          </DialogTitle>
          <DialogDescription>Update test details below.</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-test-title">Title</Label>
            <Input
              id="edit-test-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Mathematics Midterm Exam 2026"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="edit-test-subject">Subject</Label>
              <Input
                ref={subjectInputRef}
                id="edit-test-subject"
                placeholder="Search subject by name..."
                value={subjectSearch}
                onChange={(e) => handleSubjectInputChange(e.target.value)}
                onFocus={() => {
                  setShowSubjectDropdown(true);
                  if (subjectInputRef.current) {
                    const rect = subjectInputRef.current.getBoundingClientRect();
                    setSubjectDropdownPos({ top: rect.bottom + 4, left: rect.left, width: rect.width });
                  }
                }}
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
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="edit-test-coach">Coach</Label>
              <Input
                ref={coachInputRef}
                id="edit-test-coach"
                placeholder="Search coach by name..."
                value={coachSearch}
                onChange={(e) => handleCoachInputChange(e.target.value)}
                onFocus={() => {
                  setShowCoachDropdown(true);
                  if (coachInputRef.current) {
                    const rect = coachInputRef.current.getBoundingClientRect();
                    setCoachDropdownPos({ top: rect.bottom + 4, left: rect.left, width: rect.width });
                  }
                }}
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
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="edit-test-duration">Duration (minutes)</Label>
              <Input
                id="edit-test-duration"
                type="number"
                min={1}
                value={duration}
                onChange={(e) => setDuration(Number(e.target.value))}
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="edit-test-exam-date">Exam Date</Label>
              <Input
                id="edit-test-exam-date"
                type="date"
                value={examDate}
                onChange={(e) => setExamDate(e.target.value)}
              />
            </div>
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
