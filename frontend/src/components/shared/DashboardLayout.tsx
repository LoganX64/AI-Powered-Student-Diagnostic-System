import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { DashboardSidebar } from "@/components/shared/DashboardSidebar";
import { DashboardHeader } from "@/components/shared/DashboardHeader";
import { SuperAdminLayout } from "@/components/super-admin/SuperAdminLayout";

interface DashboardLayoutProps {
  title?: string;
  children: React.ReactNode;
  /**
   * Which sidebar/header chrome to use. Pages shared between the admin/coach
   * dashboard and the super-admin panel pass "super-admin" so a super-admin does
   * not land in admin navigation.
   */
  variant?: "admin" | "super-admin";
}

export function DashboardLayout({ title, children, variant = "admin" }: DashboardLayoutProps) {
  if (variant === "super-admin") {
    return <SuperAdminLayout title={title}>{children}</SuperAdminLayout>;
  }

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