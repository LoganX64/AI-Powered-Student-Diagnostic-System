import { useState, useEffect, useCallback, useRef } from "react";
import { toast } from "sonner";
import { XIcon, PlusIcon, ChevronDownIcon, CheckIcon } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { createSubject, getSubjects } from "@/services/dashboard.service";
import type { Subject } from "@/services/types";

type SubjectPickerProps = {
  selected: number[];
  onChange: (ids: number[]) => void;
  error?: string;
};

export function SubjectPicker({ selected, onChange, error }: SubjectPickerProps) {
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [search, setSearch] = useState("");
  const [creating, setCreating] = useState(false);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const fetchSubjects = useCallback(async () => {
    try {
      const res = await getSubjects({ limit: 200 });
      setSubjects(res.data ?? []);
    } catch {
      // silently ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSubjects();
  }, [fetchSubjects]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const filtered = subjects.filter(
    (s) =>
      !selected.includes(s.subject_id) &&
      s.name.toLowerCase().includes(search.toLowerCase())
  );

  const selectedSubjects = subjects.filter((s) => selected.includes(s.subject_id));

  const showCreateOption =
    search.trim().length > 0 &&
    !subjects.some(
      (s) => s.name.toLowerCase() === search.trim().toLowerCase()
    );

  const addSubject = (id: number) => {
    if (!selected.includes(id)) {
      onChange([...selected, id]);
    }
    setSearch("");
    inputRef.current?.focus();
  };

  const remove = (id: number) => {
    onChange(selected.filter((i) => i !== id));
  };

  const handleCreate = async () => {
    const name = search.trim();
    if (!name) return;

    try {
      setCreating(true);
      const res = await createSubject({ name });
      toast.success(`Subject "${name}" created`);
      const newSubject: Subject = { subject_id: res.subject_id, name };
      setSubjects((prev) => [...prev, newSubject]);
      onChange([...selected, res.subject_id]);
      setSearch("");
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setCreating(false);
    }
  };

  const hasDropdown = open && (filtered.length > 0 || showCreateOption);

  return (
    <div className="flex flex-col gap-1.5" ref={containerRef}>
      {/* Selected subjects as removable badges */}
      {selectedSubjects.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {selectedSubjects.map((s) => (
            <Badge key={s.subject_id} variant="secondary" className="gap-1 pr-1">
              {s.name}
              <button
                type="button"
                onClick={() => remove(s.subject_id)}
                className="ml-0.5 rounded-full p-0.5 hover:bg-muted"
              >
                <XIcon className="size-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}

      {/* Search / trigger input */}
      <div className="relative">
        <Input
          ref={inputRef}
          placeholder={loading ? "Loading subjects..." : "Search or create subjects..."}
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            if (!open) setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          disabled={loading}
          className="pr-8"
        />
        <button
          type="button"
          onClick={() => setOpen(!open)}
          className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
        >
          <ChevronDownIcon className="size-4" />
        </button>
      </div>

      {/* Dropdown */}
      {hasDropdown && (
        <div className="rounded-md border bg-popover shadow-sm">
          <div className="max-h-40 overflow-y-auto p-1">
            {filtered.map((s) => (
              <button
                key={s.subject_id}
                type="button"
                onClick={() => addSubject(s.subject_id)}
                className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground"
              >
                <CheckIcon className="size-3.5 text-muted-foreground" />
                {s.name}
              </button>
            ))}

            {showCreateOption && (
              <button
                type="button"
                onClick={handleCreate}
                disabled={creating}
                className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-primary hover:bg-accent hover:text-accent-foreground"
              >
                <PlusIcon className="size-3.5" />
                {creating ? "Creating..." : `Create "${search.trim()}"`}
              </button>
            )}
          </div>
        </div>
      )}

      {error && <p className="text-sm text-destructive">{error}</p>}
    </div>
  );
}
