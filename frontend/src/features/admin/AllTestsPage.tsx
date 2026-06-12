import { useEffect, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  SearchIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  Trash2Icon,
  PencilIcon,
} from "lucide-react";
import { toast } from "sonner";
import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
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
  getTests,
  deleteTest,
  type Test,
} from "@/services/admin.service";
import { EditTestDialog } from "@/components/admin/forms/EditTestDialog";
import { formatDateDDMMYYYY } from "@/lib/utils";

const PAGE_SIZE = 50;

export function AllTestsPage() {
  const navigate = useNavigate();
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
    } catch {}
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
      toast.success(`Test "${testTitle}" deleted`);
      setTests((prev) => prev.filter((t) => t.test_id !== testId));
      setTotal((prev) => prev - 1);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <SiteHeader title="All Tests" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">
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
              <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                <p className="text-sm text-muted-foreground">No tests found.</p>
              </div>
            ) : (
              <div className="flex flex-col gap-2">
                {tests.map((test) => (
                  <div key={test.test_id} className="rounded-lg border overflow-hidden">
                    <div
                      className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-muted/50"
                      onClick={() => navigate(`/admin/tests/${test.test_id}/questions`)}
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
                              Are you sure you want to delete{" "}
                              <span className="font-semibold">{test.title}</span>?
                              This will also delete all related questions.
                              This action cannot be undone.
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
              <div className="flex items-center justify-between pt-2">
                <p className="text-sm text-muted-foreground">
                  Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
                </p>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" disabled={offset === 0} onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}>
                    <ChevronLeftIcon className="size-4" /> Prev
                  </Button>
                  <Button variant="outline" size="sm" disabled={offset + PAGE_SIZE >= total} onClick={() => setOffset((o) => o + PAGE_SIZE)}>
                    Next <ChevronRightIcon className="size-4" />
                  </Button>
                </div>
              </div>
            )}
          </div>
        </div>
      </SidebarInset>

      <EditTestDialog
        test={editingTest}
        open={editingTest !== null}
        onOpenChange={(open) => { if (!open) setEditingTest(null); }}
        onUpdated={() => fetchTests(offset, search)}
      />
    </SidebarProvider>
  );
}
