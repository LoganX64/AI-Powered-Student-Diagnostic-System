import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Badge } from "@/components/ui/badge";
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
import { getAssignments, type Assignment } from "@/services/dashboard.service";
import { formatDateDDMMYYYY } from "@/lib/utils";

const PAGE_SIZE = 50;

export function TestDetailsPage() {
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";

  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loaded, setLoaded] = useState(false);
  const [filter, setFilter] = useState<"active" | "all">("active");

  useEffect(() => {
    getAssignments({
      limit: PAGE_SIZE,
      offset,
      status: filter === "active" ? "active" : undefined,
    })
      .then((res) => {
        setAssignments(res.data ?? []);
        setTotal(res.total);
      })
      .catch(() => setAssignments([]))
      .finally(() => setLoaded(true));
  }, [offset, filter]);

  return (
    <DashboardLayout title="Test Details">
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">Assigned Tests</h2>
          <div className="flex items-center gap-2">
            <div className="flex rounded-md border p-0.5 text-sm">
              <button
                type="button"
                onClick={() => {
                  setFilter("active");
                  setOffset(0);
                }}
                className={`rounded px-2.5 py-1 ${filter === "active" ? "bg-secondary font-medium" : "text-muted-foreground"}`}
              >
                Active
              </button>
              <button
                type="button"
                onClick={() => {
                  setFilter("all");
                  setOffset(0);
                }}
                className={`rounded px-2.5 py-1 ${filter === "all" ? "bg-secondary font-medium" : "text-muted-foreground"}`}
              >
                All
              </button>
            </div>
            <Badge variant="secondary">{total}</Badge>
          </div>
        </div>

        {!loaded ? (
          <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">Loading…</p>
          </div>
        ) : assignments.length === 0 ? (
          <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">
              {filter === "active"
                ? "No active (unsubmitted) assigned tests."
                : "No tests have been assigned yet."}
            </p>
          </div>
        ) : (
          <div className="rounded-lg border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Test Title</TableHead>
                  <TableHead>Student</TableHead>
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
                    <TableCell>
                      {a.student_name}
                      <span className="ml-2 font-mono text-xs text-muted-foreground">
                        {a.student_code}
                      </span>
                    </TableCell>
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
