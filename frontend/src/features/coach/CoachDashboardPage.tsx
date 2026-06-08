import { CoachSidebar } from "../../components/coach/sidebar";
import { CoachSiteHeader } from "../../components/coach/site-header";
import { CoachSectionCards } from "../../components/coach/section-cards";
import { CoachChart } from "../../components/coach/chart";
import { CoachStudentTable } from "../../components/coach/student-table";
import { SidebarInset, SidebarProvider } from "../../components/ui/sidebar";

export function CoachDashboardPage() {
  return (
    <SidebarProvider>
      <CoachSidebar />
      <SidebarInset>
        <CoachSiteHeader title="Dashboard" />
        <div className="flex flex-1 flex-col">
          <div className="@container/main flex flex-1 flex-col gap-2">
            <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
              <CoachSectionCards />
              <div className="px-4 lg:px-6">
                <CoachChart />
              </div>
              <CoachStudentTable />
            </div>
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
