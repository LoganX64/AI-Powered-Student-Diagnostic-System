import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Building2, Users, CreditCard, IndianRupee, Search } from "lucide-react";
import { Area, AreaChart, CartesianGrid, XAxis } from "recharts";
import { SuperAdminLayout } from "@/components/super-admin/SuperAdminLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Pagination, PaginationContent, PaginationItem, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import { getGlobalStats, getTenants, getPlans, type GlobalStats, type Tenant, type Plan } from "@/services/super-admin.service";

const PAGE_SIZE = 10;

const revenueData = [
  { month: "Jan", revenue: 12400 },
  { month: "Feb", revenue: 18900 },
  { month: "Mar", revenue: 24500 },
  { month: "Apr", revenue: 31200 },
  { month: "May", revenue: 38700 },
  { month: "Jun", revenue: 49900 },
];

const revenueChartConfig = {
  revenue: {
    label: "Revenue",
    color: "var(--primary)",
  },
} satisfies ChartConfig;

const planBadgeVariant: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  free: "secondary",
  starter: "default",
  professional: "default",
  enterprise: "default",
};

const planBadgeColor: Record<string, string> = {
  starter: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
  professional: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
  enterprise: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
};

function getPlanName(planId: number | null, plans: Plan[]): string {
  if (!planId) return "Free";
  const plan = plans.find((p) => p.id === planId);
  return plan?.name ?? "Free";
}

function getPlanSlug(planId: number | null, plans: Plan[]): string {
  if (!planId) return "free";
  const plan = plans.find((p) => p.id === planId);
  return plan?.slug ?? "free";
}

export function SuperAdminDashboardPage() {
  const navigate = useNavigate();
  const [stats, setStats] = useState<GlobalStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(true);
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [planFilter, setPlanFilter] = useState("all");
  const [plans, setPlans] = useState<Plan[]>([]);
  const [tableLoading, setTableLoading] = useState(true);
  const searchDebounce = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const [statsData, plansData] = await Promise.all([
          getGlobalStats(),
          getPlans(),
        ]);
        if (!cancelled) {
          setStats(statsData);
          setPlans(plansData.data ?? []);
        }
      } catch (err) {
        toast.error((err as Error).message);
      } finally {
        if (!cancelled) setStatsLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  const fetchTenants = useCallback(async (off: number, searchTerm: string, plan: string, shouldAbort?: () => boolean) => {
    setTableLoading(true);
    try {
      const res = await getTenants({
        limit: PAGE_SIZE,
        offset: off,
        search: searchTerm || undefined,
        plan: plan === "all" ? undefined : plan,
      });
      if (shouldAbort?.()) return;
      setTenants(res.data ?? []);
      setTotal(res.total);
    } catch (err) {
      if (!shouldAbort?.()) toast.error((err as Error).message);
    } finally {
      if (!shouldAbort?.()) setTableLoading(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    fetchTenants(offset, search, planFilter, () => cancelled);
    return () => { cancelled = true; };
  }, [offset, search, planFilter, fetchTenants]);

  useEffect(() => {
    if (searchDebounce.current) clearTimeout(searchDebounce.current);
    searchDebounce.current = setTimeout(() => {
      setSearch(searchInput);
      setOffset(0);
    }, 300);
    return () => clearTimeout(searchDebounce.current);
  }, [searchInput]);

  const handlePlanFilterChange = (value: string) => {
    setPlanFilter(value);
    setOffset(0);
  };

  const statCards = [
    { key: "tenants" as const, title: "Total Tenants", icon: Building2 },
    { key: "free_tenants" as const, title: "Free Users", icon: Users },
    { key: "paid_tenants" as const, title: "Paid Users", icon: CreditCard },
    { key: "revenue" as const, title: "Revenue", icon: IndianRupee, isCurrency: true },
  ];

  return (
    <SuperAdminLayout title="Super Admin Dashboard">
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {statCards.map((card) => (
          <Card key={card.key}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">{card.title}</CardTitle>
              <card.icon className="size-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              {statsLoading ? (
                <Skeleton className="h-8 w-20" />
              ) : (
                <div className="text-2xl font-bold">
                  {card.isCurrency
                    ? `₹${(stats?.[card.key] ?? 0).toLocaleString("en-IN")}`
                    : (stats?.[card.key] ?? 0).toLocaleString()}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Revenue Overview</CardTitle>
        </CardHeader>
        <CardContent className="px-2 pt-4 sm:px-6 sm:pt-6">
          <ChartContainer config={revenueChartConfig} className="aspect-auto h-[250px] w-full">
            <AreaChart data={revenueData}>
              <defs>
                <linearGradient id="fillRevenue" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--color-revenue)" stopOpacity={0.8} />
                  <stop offset="95%" stopColor="var(--color-revenue)" stopOpacity={0.1} />
                </linearGradient>
              </defs>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="month"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                minTickGap={32}
              />
              <ChartTooltip
                cursor={false}
                content={
                  <ChartTooltipContent
                    formatter={(value) => [`₹${Number(value).toLocaleString("en-IN")}`, "Revenue"]}
                    indicator="dot"
                  />
                }
              />
              <Area
                dataKey="revenue"
                type="natural"
                fill="url(#fillRevenue)"
                stroke="var(--color-revenue)"
              />
            </AreaChart>
          </ChartContainer>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Tenants</CardTitle>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="absolute left-2 top-2.5 size-4 text-muted-foreground" />
              <Input
                placeholder="Search tenants..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                className="pl-8 w-48"
              />
            </div>
            <Select value={planFilter} onValueChange={handlePlanFilterChange}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="All Plans" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Plans</SelectItem>
                <SelectItem value="free">Free</SelectItem>
                <SelectItem value="starter">Starter</SelectItem>
                <SelectItem value="professional">Professional</SelectItem>
                <SelectItem value="enterprise">Enterprise</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Plan</TableHead>
                  <TableHead className="text-center">Students</TableHead>
                  <TableHead className="text-center">Coaches</TableHead>
                  <TableHead className="text-center">Users</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Joined At</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tableLoading ? (
                  <TableRow>
                    <TableCell colSpan={8} className="text-center py-8">Loading...</TableCell>
                  </TableRow>
                ) : tenants.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">No tenants found</TableCell>
                  </TableRow>
                ) : tenants.map((t) => {
                  const planSlug = getPlanSlug(t.plan_id, plans);
                  return (
                    <TableRow
                      key={t.id}
                      className="cursor-pointer hover:bg-muted/50"
                      onClick={() => navigate(`/super-admin/tenants/${t.id}`)}
                    >
                      <TableCell className="font-mono text-sm text-muted-foreground">{t.id}</TableCell>
                      <TableCell className="font-medium">{t.name}</TableCell>
                      <TableCell>
                        <Badge
                          variant={planBadgeVariant[planSlug] ?? "secondary"}
                          className={planBadgeColor[planSlug] ?? ""}
                        >
                          {getPlanName(t.plan_id, plans)}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-center">{t.student_count}</TableCell>
                      <TableCell className="text-center">{t.coach_count}</TableCell>
                      <TableCell className="text-center">{t.user_count}</TableCell>
                      <TableCell>
                        <Badge variant={t.suspended_at ? "destructive" : "default"}>
                          {t.suspended_at ? "Suspended" : "Active"}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(t.created_at).toLocaleDateString("en-IN", {
                          day: "numeric",
                          month: "short",
                          year: "numeric",
                        })}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
          {total > PAGE_SIZE && (
            <Pagination className="mt-4">
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
        </CardContent>
      </Card>
    </SuperAdminLayout>
  );
}
