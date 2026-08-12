import * as React from "react";
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type ColumnFiltersState,
  type SortingState,
  type VisibilityState,
} from "@tanstack/react-table";
import {
  CircleCheckIcon,
  AlertCircleIcon,
  ClockIcon,
  EllipsisVerticalIcon,
  ChevronDownIcon,
  ChevronsLeftIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsRightIcon,
  SearchIcon,
  UsersIcon,
  GraduationCapIcon,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuCheckboxItem,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectGroup,
  SelectItem,
} from "@/components/ui/select";
import {
  Table,
  TableRow,
  TableCell,
  TableHeader,
  TableHead,
  TableBody,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Columns3Icon } from "lucide-react";
import { useDashboard, type StudentWithSQI, type CoachRow } from "@/contexts/DashboardContext";

// ─── Student Types & Data ─────────────────────────────────────────────────────

export type StudentRow = {
  id: number;
  name: string;
  email: string;
  sqiScore: number;
  status: "Passing" | "At Risk" | "Pending";
  lastAssessment: string;
  completedQuizzes: number;
};

function getStudentStatus(sqi: number): "Passing" | "At Risk" | "Pending" {
  if (sqi === 0) return "Pending";
  if (sqi < 55) return "At Risk";
  return "Passing";
}

function toStudentRows(students: StudentWithSQI[]): StudentRow[] {
  return students.map((s) => ({
    id: s.student_id,
    name: s.name,
    email: s.student_code,
    sqiScore: s.average_sqi,
    status: getStudentStatus(s.average_sqi),
    lastAssessment: s.total_tests > 0 ? `${s.total_tests} test(s)` : "—",
    completedQuizzes: s.total_tests,
  }));
}

// ─── Helper Functions ─────────────────────────────────────────────────────────

const studentStatusIcon = (status: StudentRow["status"]) => {
  if (status === "Passing") return <CircleCheckIcon className="size-3.5 fill-green-500 dark:fill-green-400" />;
  if (status === "At Risk") return <AlertCircleIcon className="size-3.5 text-destructive" />;
  return <ClockIcon className="size-3.5 text-muted-foreground" />;
};

const studentStatusVariant = (status: StudentRow["status"]): "outline" | "secondary" => {
  return status === "At Risk" ? "outline" : "secondary";
};

const coachStatusIcon = (status: CoachRow["status"]) => {
  if (status === "Active") return <CircleCheckIcon className="size-3.5 fill-green-500 dark:fill-green-400" />;
  return <ClockIcon className="size-3.5 text-muted-foreground" />;
};

const coachStatusVariant = (status: CoachRow["status"]): "outline" | "secondary" => {
  return status === "Inactive" ? "outline" : "secondary";
};

// ─── Column Definitions ───────────────────────────────────────────────────────

const studentColumns: ColumnDef<StudentRow>[] = [
  {
    accessorKey: "name",
    header: "Student",
    cell: ({ row }) => (
      <div className="flex flex-col">
        <span className="font-medium">{row.original.name}</span>
        <span className="text-xs text-muted-foreground">{row.original.email}</span>
      </div>
    ),
    enableHiding: false,
  },
  {
    accessorKey: "sqiScore",
    header: "SQI Score",
    cell: ({ row }) => {
      const score = row.original.sqiScore;
      if (row.original.status === "Pending") return <span className="text-muted-foreground text-sm">—</span>;
      return (
        <div className="flex items-center gap-2">
          <div className="h-2 w-16 rounded-full bg-muted overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${score >= 70 ? "bg-green-500" : score >= 55 ? "bg-yellow-500" : "bg-destructive"}`}
              style={{ width: `${score}%` }}
            />
          </div>
          <span className="tabular-nums text-sm font-medium">{score}</span>
        </div>
      );
    },
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => (
      <Badge variant={studentStatusVariant(row.original.status)} className="px-1.5 text-muted-foreground gap-1">
        {studentStatusIcon(row.original.status)}
        {row.original.status}
      </Badge>
    ),
  },
  {
    accessorKey: "completedQuizzes",
    header: "Quizzes",
    cell: ({ row }) => (
      <span className="tabular-nums text-sm">{row.original.completedQuizzes}</span>
    ),
  },
  {
    accessorKey: "lastAssessment",
    header: "Last Assessment",
    cell: ({ row }) => (
      <span className="text-sm text-muted-foreground">{row.original.lastAssessment}</span>
    ),
  },
  {
    id: "actions",
    cell: () => (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            className="flex size-8 text-muted-foreground data-[state=open]:bg-muted"
            size="icon"
          >
            <EllipsisVerticalIcon />
            <span className="sr-only">Open menu</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-40">
          <DropdownMenuItem>View Profile</DropdownMenuItem>
          <DropdownMenuItem>View Results</DropdownMenuItem>
          <DropdownMenuItem>Send Message</DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive">Remove Student</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    ),
  },
];

const coachColumns: ColumnDef<CoachRow>[] = [
  {
    accessorKey: "name",
    header: "Coach",
    cell: ({ row }) => (
      <div className="flex flex-col">
        <span className="font-medium">{row.original.name}</span>
        <span className="text-xs text-muted-foreground">{row.original.email}</span>
      </div>
    ),
    enableHiding: false,
  },
  {
    accessorKey: "studentsCount",
    header: "Students",
    cell: ({ row }) => (
      <span className="tabular-nums text-sm">{row.original.studentsCount}</span>
    ),
  },
  {
    accessorKey: "avgStudentSqi",
    header: "Avg. Student SQI",
    cell: ({ row }) => {
      const score = row.original.avgStudentSqi;
      if (score === 0) return <span className="text-muted-foreground text-sm">—</span>;
      return (
        <div className="flex items-center gap-2">
          <div className="h-2 w-16 rounded-full bg-muted overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${score >= 70 ? "bg-green-500" : score >= 55 ? "bg-yellow-500" : "bg-destructive"}`}
              style={{ width: `${score}%` }}
            />
          </div>
          <span className="tabular-nums text-sm font-medium">{score}</span>
        </div>
      );
    },
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => (
      <Badge variant={coachStatusVariant(row.original.status)} className="px-1.5 text-muted-foreground gap-1">
        {coachStatusIcon(row.original.status)}
        {row.original.status}
      </Badge>
    ),
  },
  {
    accessorKey: "joinedDate",
    header: "Joined",
    cell: ({ row }) => {
      const jd = row.original.joinedDate;
      const d = jd && !isNaN(Date.parse(jd)) ? new Date(jd) : null;
      return (
        <span className="text-sm text-muted-foreground">
          {d ? d.toLocaleDateString() : "—"}
        </span>
      );
    },
  },
  {
    id: "actions",
    cell: () => (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            className="flex size-8 text-muted-foreground data-[state=open]:bg-muted"
            size="icon"
          >
            <EllipsisVerticalIcon />
            <span className="sr-only">Open menu</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-40">
          <DropdownMenuItem>View Profile</DropdownMenuItem>
          <DropdownMenuItem>View Students</DropdownMenuItem>
          <DropdownMenuItem>Send Message</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    ),
  },
];

// ─── Reusable Table Component ─────────────────────────────────────────────────

function DataTable<T>({
  data,
  columns,
  searchPlaceholder,
}: {
  data: T[];
  columns: ColumnDef<T>[];
  searchPlaceholder: string;
}) {
  const [rowSelection, setRowSelection] = React.useState({});
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({});
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [pagination, setPagination] = React.useState({ pageIndex: 0, pageSize: 10 });
  const [globalFilter, setGlobalFilter] = React.useState("");

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    data,
    columns,
    state: { sorting, columnVisibility, rowSelection, columnFilters, pagination, globalFilter },
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  return (
    <div className="w-full">
      <div className="flex items-center justify-between pb-4 gap-3">
        <div className="relative flex-1 max-w-sm">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
          <Input
            placeholder={searchPlaceholder}
            value={globalFilter}
            onChange={(e) => setGlobalFilter(e.target.value)}
            className="pl-9"
          />
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm">
              <Columns3Icon data-icon="inline-start" />
              Columns
              <ChevronDownIcon data-icon="inline-end" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-36">
            {table
              .getAllColumns()
              .filter((col) => typeof col.accessorFn !== "undefined" && col.getCanHide())
              .map((col) => (
                <DropdownMenuCheckboxItem
                  key={col.id}
                  className="capitalize"
                  checked={col.getIsVisible()}
                  onCheckedChange={(value) => col.toggleVisibility(!!value)}
                >
                  {col.id}
                </DropdownMenuCheckboxItem>
              ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <div className="overflow-hidden rounded-lg border">
        <Table>
          <TableHeader className="sticky top-0 z-10 bg-muted">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id} colSpan={header.colSpan}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} data-state={row.getIsSelected() && "selected"}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center">
                  No results found.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-between pt-4">
        <div className="hidden flex-1 text-sm text-muted-foreground lg:flex">
          {table.getFilteredRowModel().rows.length} row(s) total
        </div>
        <div className="flex w-full items-center gap-8 lg:w-fit">
          <div className="hidden items-center gap-2 lg:flex">
            <Label htmlFor="rows-per-page" className="text-sm font-medium">
              Rows per page
            </Label>
            <Select
              value={`${table.getState().pagination.pageSize}`}
              onValueChange={(value) => table.setPageSize(Number(value))}
            >
              <SelectTrigger size="sm" className="w-20" id="rows-per-page">
                <SelectValue placeholder={table.getState().pagination.pageSize} />
              </SelectTrigger>
              <SelectContent side="top">
                <SelectGroup>
                  {[10, 20, 30].map((size) => (
                    <SelectItem key={size} value={`${size}`}>{size}</SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
          <div className="flex w-fit items-center justify-center text-sm font-medium">
            Page {table.getState().pagination.pageIndex + 1} of {table.getPageCount()}
          </div>
          <div className="ml-auto flex items-center gap-2 lg:ml-0">
            <Button
              variant="outline"
              className="hidden h-8 w-8 p-0 lg:flex"
              onClick={() => table.setPageIndex(0)}
              disabled={!table.getCanPreviousPage()}
            >
              <span className="sr-only">Go to first page</span>
              <ChevronsLeftIcon />
            </Button>
            <Button
              variant="outline"
              className="size-8"
              size="icon"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
            >
              <span className="sr-only">Go to previous page</span>
              <ChevronLeftIcon />
            </Button>
            <Button
              variant="outline"
              className="size-8"
              size="icon"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
            >
              <span className="sr-only">Go to next page</span>
              <ChevronRightIcon />
            </Button>
            <Button
              variant="outline"
              className="hidden size-8 lg:flex"
              size="icon"
              onClick={() => table.setPageIndex(table.getPageCount() - 1)}
              disabled={!table.getCanNextPage()}
            >
              <span className="sr-only">Go to last page</span>
              <ChevronsRightIcon />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Main Component ───────────────────────────────────────────────────────────

export function DashboardTable() {
  const { studentsWithSQI, coachRows, loading, role } = useDashboard();
  const studentData = React.useMemo(() => toStudentRows(studentsWithSQI), [studentsWithSQI]);

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <p className="text-muted-foreground">Loading students...</p>
      </div>
    );
  }

  if (role === "admin") {
    return (
      <Tabs defaultValue="students" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="students" className="flex items-center gap-2">
            <GraduationCapIcon className="size-4" />
            Students
          </TabsTrigger>
          <TabsTrigger value="coaches" className="flex items-center gap-2">
            <UsersIcon className="size-4" />
            Coaches
          </TabsTrigger>
        </TabsList>
        <TabsContent value="students" className="mt-4">
          <DataTable
            data={studentData}
            columns={studentColumns}
            searchPlaceholder="Search students..."
          />
        </TabsContent>
        <TabsContent value="coaches" className="mt-4">
          <DataTable
            data={coachRows}
            columns={coachColumns}
            searchPlaceholder="Search coaches..."
          />
        </TabsContent>
      </Tabs>
    );
  }

  return (
    <DataTable
      data={studentData}
      columns={studentColumns}
      searchPlaceholder="Search students..."
    />
  );
}
