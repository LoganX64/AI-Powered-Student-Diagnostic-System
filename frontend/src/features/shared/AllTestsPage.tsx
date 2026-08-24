import { useEffect, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  SearchIcon,
  Trash2Icon,
  PencilIcon,
  EyeIcon,
} from "lucide-react";
import { toast } from "sonner";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Pagination, PaginationContent, PaginationItem, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  getTests,
  deleteTest,
  type Test,
} from "@/services/dashboard.service";
import { EditTestDialog } from "@/components/admin/forms/EditTestDialog";
import { formatDateDDMMYYYY } from "@/lib/utils";

const PAGE_SIZE = 50;

export function AllTestsPage() {
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";
  const isAdmin = role === "admin";

  const [tests, setTests] = useState<Test[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");

  const [editingTest, setEditingTest] = useState<Test | null>(null);

  const fetchTests = useCallback(async (off: number, searchTerm: string) => {
    try {
      const res = await getTests({ limit: PAGE_SIZE, offset: off, search: searchTerm || undefined });
      setTests(res.data ?? []);
      setTotal(res.total);
    } catch (err) {
      void err;
    }
  }, []);

  useEffect(() => {
    fetchTests(offset, search);
  }, [offset, search, fetchTests]);

  const handleSearch = () => {
    setOffset(0);
    setSearch(searchInput);
  };

  const handleDeleteTest = async (testId: number, testTitle: string) => {
    try {
      await deleteTest(testId);
      toast.success(`Test "${testTitle}" deactivated`);
      setTests((prev) => prev.filter((t) => t.test_id !== testId));
      setTotal((prev) => prev - 1);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  // Coach view: Table layout
  if (!isAdmin) {
    return (
      <DashboardLayout title="All Tests">
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-sm">
            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
            <Input
              placeholder="Search by test title..."
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              className="pl-9"
            />
          </div>
          <Button variant="outline" onClick={handleSearch}>Search</Button>
        </div>

        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold">All Tests</h2>
            <Badge variant="secondary">{total}</Badge>
          </div>

          {tests.length === 0 ? (
            <div className="flex h-40 flex-col items-center justify-center gap-3 rounded-lg border border-dashed">
              <p className="text-sm text-muted-foreground">No tests created yet.</p>
              <Button variant="outline" size="sm" onClick={() => navigate(`${prefix}/tests`)}>
                Create Your First Test
              </Button>
            </div>
          ) : (
            <div className="rounded-lg border overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-16">ID</TableHead>
                    <TableHead>Title</TableHead>
                    <TableHead>Subject</TableHead>
                    <TableHead>Exam Date</TableHead>
                    <TableHead>Duration</TableHead>
                    <TableHead className="w-20 text-right">Action</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tests.map((test) => (
                    <TableRow
                      key={test.test_id}
                      className="cursor-pointer hover:bg-muted/50"
                      onClick={() => navigate(`${prefix}/tests/${test.test_id}/questions`)}
                    >
                      <TableCell className="font-mono text-sm text-muted-foreground">
                        {test.test_id}
                      </TableCell>
                      <TableCell className="font-medium">{test.title}</TableCell>
                      <TableCell>{test.subject_name || `#${test.subject_id}`}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {test.exam_date ? formatDateDDMMYYYY(test.exam_date) : "—"}
                      </TableCell>
                      <TableCell className="text-sm">{test.duration}m</TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-8 text-muted-foreground hover:text-foreground"
                          aria-label={`View ${test.title}`}
                          onClick={(e) => { e.stopPropagation(); navigate(`${prefix}/tests/${test.test_id}/questions`); }}
                        >
                          <EyeIcon className="size-4" />
                        </Button>
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

  // Admin view: Card layout with edit/delete
  return (
    <DashboardLayout title="All Tests">
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            placeholder="Search by test title..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="pl-9"
          />
        </div>
        <Button variant="outline" onClick={handleSearch}>Search</Button>
      </div>

      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">All Tests</h2>
          <Badge variant="secondary">{total}</Badge>
        </div>

        {tests.length === 0 ? (
          <div className="flex h-40 flex-col items-center justify-center gap-3 rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">No tests created yet.</p>
            <Button variant="outline" size="sm" onClick={() => navigate(`${prefix}/tests`)}>
              Create Your First Test
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {tests.map((test) => (
              <div key={test.test_id} className="rounded-lg border overflow-hidden">
                <div
                  className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-muted/50"
                  onClick={() => navigate(`${prefix}/tests/${test.test_id}/questions`)}
                >
                  <span className="font-medium flex-1">{test.title}</span>
                  {test.exam_date && (
                    <Badge variant="outline" className="hidden sm:inline-flex">Exam: {formatDateDDMMYYYY(test.exam_date)}</Badge>
                  )}
                  <Badge variant="secondary" className="hidden sm:inline-flex">{test.subject_name || `#${test.subject_id}`}</Badge>
                  <Badge variant="outline" className="hidden sm:inline-flex">{test.coach_name || `#${test.coach_id}`}</Badge>
                  <span className="text-sm text-muted-foreground">{test.duration}m</span>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-8 text-muted-foreground hover:text-foreground"
                    aria-label={`Edit ${test.title}`}
                    onClick={(e) => { e.stopPropagation(); setEditingTest(test); }}
                  >
                    <PencilIcon className="size-4" />
                  </Button>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="size-8 text-muted-foreground hover:text-destructive"
                        aria-label={`Delete ${test.title}`}
                        onClick={(e) => e.stopPropagation()}
                      >
                        <Trash2Icon className="size-4" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete Test</AlertDialogTitle>
                        <AlertDialogDescription>
                          Are you sure you want to deactivate{" "}
                          <span className="font-semibold">{test.title}</span>?
                          This test will be deactivated. Students who attempted it will keep their data.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={(e) => { e.stopPropagation(); handleDeleteTest(test.test_id, test.title); }}
                          className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                          Delete
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </div>
            ))}
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

      <EditTestDialog
        test={editingTest}
        open={editingTest !== null}
        onOpenChange={(open) => { if (!open) setEditingTest(null); }}
        onUpdated={() => fetchTests(offset, search)}
      />
    </DashboardLayout>
  );
}
