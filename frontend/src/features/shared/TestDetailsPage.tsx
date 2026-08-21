import { useEffect, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SearchableSelect } from "@/components/ui/searchable-select";
import { getAssignments, getSubjects, getCoaches, type Assignment, type Subject, type Coach } from "@/services/dashboard.service";
import { formatDateDDMMYYYY } from "@/lib/utils";

const PAGE_SIZE = 50;

function getYearOptions(): string[] {
  const current = new Date().getFullYear();
  const years: string[] = [];
  for (let y = current; y >= current - 5; y--) {
    years.push(String(y));
  }
  return years;
}

export function TestDetailsPage() {
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";

  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loaded, setLoaded] = useState(false);
  const [filter, setFilter] = useState<"active" | "all" | "submitted">("active");

  const [search, setSearch] = useState("");
  const [year, setYear] = useState("");
  const [subjectId, setSubjectId] = useState("");
  const [coachId, setCoachId] = useState("");

  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [coaches, setCoaches] = useState<Coach[]>([]);

  const yearOptions = getYearOptions();

  useEffect(() => {
    getSubjects({ limit: 200 }).then((res) => setSubjects(res.data ?? [])).catch(() => {});
    if (role === "admin") {
      getCoaches({ limit: 200 }).then((res) => setCoaches(res.data ?? [])).catch(() => {});
    }
  }, [role]);

  const fetchAssignments = useCallback(async () => {
    setLoaded(false);
    try {
      const res = await getAssignments({
        limit: PAGE_SIZE,
        offset,
        status: filter === "all" ? undefined : filter,
        search: search || undefined,
        year: year || undefined,
        subject_id: subjectId ? Number(subjectId) : undefined,
        coach_id: coachId ? Number(coachId) : undefined,
      });
      setAssignments(res.data ?? []);
      setTotal(res.total);
    } catch {
      setAssignments([]);
    } finally {
      setLoaded(true);
    }
  }, [offset, filter, search, year, subjectId, coachId]);

  useEffect(() => {
    fetchAssignments();
  }, [fetchAssignments]);

  const resetFilters = () => {
    setSearch("");
    setYear("");
    setSubjectId("");
    setCoachId("");
    setFilter("active");
    setOffset(0);
  };

  return (
    <DashboardLayout title="Test Details">
      <div className="flex flex-col gap-4">
        {/* Filters row */}
        <div className="flex flex-col gap-3 rounded-lg border p-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">Filters</h3>
            <button
              type="button"
              onClick={resetFilters}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Reset all
            </button>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Search</Label>
              <Input
                placeholder="Student, test, or subject..."
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Year</Label>
              <Select value={year} onValueChange={(v) => { setYear(v === "all" ? "" : v); setOffset(0); }}>
                <SelectTrigger>
                  <SelectValue placeholder="All years" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All years</SelectItem>
                  {yearOptions.map((y) => (
                    <SelectItem key={y} value={y}>{y}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Subject</Label>
              <SearchableSelect
                options={[
                  { label: "All subjects", value: "" },
                  ...subjects.map((s) => ({ label: s.name, value: String(s.subject_id) })),
                ]}
                value={subjectId}
                onChange={(v) => { setSubjectId(v); setOffset(0); }}
                placeholder="Search subjects..."
              />
            </div>
            {role === "admin" && (
              <div className="flex flex-col gap-1.5">
                <Label className="text-xs">Coach</Label>
                <SearchableSelect
                  options={[
                    { label: "All coaches", value: "" },
                    ...coaches.map((c) => ({ label: c.name, value: String(c.coach_id) })),
                  ]}
                  value={coachId}
                  onChange={(v) => { setCoachId(v); setOffset(0); }}
                  placeholder="Search coaches..."
                />
              </div>
            )}
          </div>
        </div>

        {/* Status filter + count */}
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">Assigned Tests</h2>
          <div className="flex items-center gap-2">
            <div className="flex rounded-md border p-0.5 text-sm">
              <button
                type="button"
                onClick={() => { setFilter("active"); setOffset(0); }}
                className={`rounded px-2.5 py-1 ${filter === "active" ? "bg-secondary font-medium" : "text-muted-foreground"}`}
              >
                Active
              </button>
              <button
                type="button"
                onClick={() => { setFilter("submitted"); setOffset(0); }}
                className={`rounded px-2.5 py-1 ${filter === "submitted" ? "bg-secondary font-medium" : "text-muted-foreground"}`}
              >
                Submitted
              </button>
              <button
                type="button"
                onClick={() => { setFilter("all"); setOffset(0); }}
                className={`rounded px-2.5 py-1 ${filter === "all" ? "bg-secondary font-medium" : "text-muted-foreground"}`}
              >
                All
              </button>
            </div>
            <Badge variant="secondary">{total}</Badge>
          </div>
        </div>

        {/* Table */}
        {!loaded ? (
          <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">Loading…</p>
          </div>
        ) : assignments.length === 0 ? (
          <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">
              {filter === "active"
                ? "No active (unsubmitted) assigned tests."
                : filter === "submitted"
                ? "No submitted assigned tests."
                : "No tests have been assigned yet."}
            </p>
          </div>
        ) : (
          <div className="rounded-lg border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Test Title</TableHead>
                  <TableHead>Subject</TableHead>
                  <TableHead>Student</TableHead>
                  <TableHead>Coach</TableHead>
                  <TableHead className="w-36">Status</TableHead>
                  <TableHead>Assigned At</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {assignments.map((a) => (
                  <TableRow
                    key={a.id}
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => navigate(`${prefix}/students/${a.student_id}/assignments/${a.id}`)}
                  >
                    <TableCell className="font-medium">{a.test_title}</TableCell>
                    <TableCell>{a.subject_name || "—"}</TableCell>
                    <TableCell>
                      {a.student_name}
                      <span className="ml-2 font-mono text-xs text-muted-foreground">
                        {a.student_code}
                      </span>
                    </TableCell>
                    <TableCell>{a.coach_name || "—"}</TableCell>
                    <TableCell>
                      <Badge variant={a.status === "submitted" ? "default" : "secondary"}>
                        {a.status === "submitted" ? "Submitted" : "Active"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDateDDMMYYYY(a.assigned_at)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        {/* Pagination */}
        {total > PAGE_SIZE && (
          <Pagination>
            <PaginationContent className="flex items-center justify-between w-full">
              <p className="text-sm text-muted-foreground">
                Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
              </p>
              <div className="flex gap-2">
                <PaginationItem>
                  <PaginationPrevious
                    onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
                    className={offset === 0 ? "pointer-events-none opacity-50" : ""}
                  />
                </PaginationItem>
                <PaginationItem>
                  <PaginationNext
                    onClick={() => setOffset((o) => o + PAGE_SIZE)}
                    className={offset + PAGE_SIZE >= total ? "pointer-events-none opacity-50" : ""}
                  />
                </PaginationItem>
              </div>
            </PaginationContent>
          </Pagination>
        )}
      </div>
    </DashboardLayout>
  );
}
