import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Building2, Users, GraduationCap, UserCheck } from "lucide-react";
import { SuperAdminLayout } from "@/components/super-admin/SuperAdminLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { getGlobalStats, type GlobalStats } from "@/services/super-admin.service";

const statCards = [
  { key: "tenants" as const, title: "Total Tenants", icon: Building2 },
  { key: "users" as const, title: "Total Users", icon: Users },
  { key: "students" as const, title: "Total Students", icon: GraduationCap },
  { key: "coaches" as const, title: "Total Coaches", icon: UserCheck },
];

export function SuperAdminDashboardPage() {
  const [stats, setStats] = useState<GlobalStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const data = await getGlobalStats();
        if (!cancelled) setStats(data);
      } catch (err) {
        toast.error((err as Error).message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, []);

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
              {loading ? (
                <Skeleton className="h-8 w-20" />
              ) : (
                <div className="text-2xl font-bold">{stats?.[card.key] ?? 0}</div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </SuperAdminLayout>
  );
}
