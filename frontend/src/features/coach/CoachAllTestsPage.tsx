import { useEffect, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { SearchIcon, ChevronLeftIcon, ChevronRightIcon, EyeIcon } from "lucide-react";
import { CoachSidebar } from "@/components/coach/sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { getTests, type Test } from "@/services/coach.service";

const PAGE_SIZE = 50;

export function CoachAllTestsPage() {
  const navigate = useNavigate();
  const [tests, setTests] = useState<Test[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");

  const fetchTests = useCallback(async (off: number, searchTerm: string) => {
    try {
      const res = await getTests({ limit: PAGE_SIZE, offset: off, search: searchTerm || undefined });
      setTests(res.data);
      setTotal(res.total);
    } catch {
      // silently ignore
    }
  }, []);

  useEffect(() => {
    fetchTests(offset, search);
  }, [offset, search, fetchTests]);

  const handleSearch = () => {
    setOffset(0);
    setSearch(searchInput);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  return (
    <SidebarProvider>
      <CoachSidebar />
      <SidebarInset>
        <SiteHeader title="All Tests" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

          {/* Search bar */}
          <div className="flex items-center gap-3">
            <div className="relative flex-1 max-w-sm">
              <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
              <Input
                placeholder="Search by test title..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={handleKeyDown}
                className="pl-9"
              />
            </div>
            <Button variant="outline" onClick={handleSearch}>
              Search
            </Button>
          </div>

          {/* Tests table */}
          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">All Tests</h2>
              <Badge variant="secondary">{total}</Badge>
            </div>

            {tests.length === 0 ? (
              <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                <p className="text-sm text-muted-foreground">
                  No tests found.
                </p>
              </div>
            ) : (
              <div className="rounded-lg border overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-16">ID</TableHead>
                      <TableHead>Title</TableHead>
                      <TableHead className="w-24">Subject ID</TableHead>
                      <TableHead className="w-24">Duration</TableHead>
                      <TableHead className="w-20 text-right">Action</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {tests.map((test) => (
                      <TableRow key={test.test_id}>
                        <TableCell className="font-mono text-sm text-muted-foreground">
                          {test.test_id}
                        </TableCell>
                        <TableCell className="font-medium">{test.title}</TableCell>
                        <TableCell className="text-muted-foreground">
                          {test.subject_id}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {test.duration}s
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="size-8 text-muted-foreground hover:text-foreground"
                            onClick={() => navigate(`/coach/tests/${test.test_id}`)}
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

            {/* Pagination */}
            {total > PAGE_SIZE && (
              <div className="flex items-center justify-between pt-2">
                <p className="text-sm text-muted-foreground">
                  Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={offset === 0}
                    onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
                  >
                    <ChevronLeftIcon className="size-4" /> Prev
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={offset + PAGE_SIZE >= total}
                    onClick={() => setOffset((o) => o + PAGE_SIZE)}
                  >
                    Next <ChevronRightIcon className="size-4" />
                  </Button>
                </div>
              </div>
            )}
          </div>

        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
