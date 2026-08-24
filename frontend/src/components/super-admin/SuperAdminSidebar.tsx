import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { NavUser } from "@/components/shared/nav-user";
import { NavSecondary } from "@/components/shared/nav-secondary";
import { LayoutDashboard, Building2, CreditCard, Settings2 } from "lucide-react";
import { Link, useLocation } from "react-router-dom";

const navItems = [
  { title: "Dashboard", url: "/super-admin/dashboard", icon: <LayoutDashboard className="size-4" /> },
  { title: "Tenants", url: "/super-admin/tenants", icon: <Building2 className="size-4" /> },
  { title: "Plans", url: "/super-admin/plans", icon: <CreditCard className="size-4" /> },
];

const secondaryNavItems = [
  { title: "Settings", url: "/super-admin/settings", icon: <Settings2 className="size-4" /> },
];

export function SuperAdminSidebar({ user }: { user: { name: string; email: string; avatar: string } }) {
  const location = useLocation();

  return (
    <Sidebar>
      <SidebarHeader>
        <div className="flex items-center gap-2 px-2 py-1">
          <span className="text-lg font-semibold">Super Admin Panel</span>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarMenu>
          {navItems.map((item) => (
            <SidebarMenuItem key={item.url}>
              <SidebarMenuButton asChild isActive={location.pathname === item.url}>
                <Link to={item.url}>
                  {item.icon}
                  <span>{item.title}</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
        <NavSecondary items={secondaryNavItems} className="mt-auto" />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
    </Sidebar>
  );
}
