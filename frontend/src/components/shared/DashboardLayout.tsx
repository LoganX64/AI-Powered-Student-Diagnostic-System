import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { DashboardSidebar } from "@/components/shared/DashboardSidebar";
import { DashboardHeader } from "@/components/shared/DashboardHeader";

interface DashboardLayoutProps {
  title?: string;
  children: React.ReactNode;
}

export function DashboardLayout({ title, children }: DashboardLayoutProps) {
  return (
    <SidebarProvider>
      <DashboardSidebar />
      <SidebarInset>
        <DashboardHeader title={title} />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">
          {children}
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
