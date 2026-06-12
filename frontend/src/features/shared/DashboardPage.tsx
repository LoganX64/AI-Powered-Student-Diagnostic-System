import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { DashboardSectionCards } from "@/components/shared/DashboardSectionCards";
import { DashboardChart } from "@/components/shared/DashboardChart";
import { DashboardTable } from "@/components/shared/DashboardTable";

export function DashboardPage() {
  return (
    <DashboardLayout title="Dashboard">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
          <DashboardSectionCards />
          <div className="px-4 lg:px-6">
            <DashboardChart />
          </div>
          <div className="px-4 lg:px-6">
            <DashboardTable />
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}
