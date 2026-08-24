import { useEffect, useState } from "react";
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar";
import { SuperAdminSidebar } from "./SuperAdminSidebar";
import { DashboardHeader } from "@/components/shared/DashboardHeader";
import { apiFetch } from "@/lib/api";

interface Profile {
  email: string;
  display_name: string | null;
  avatar_url: string | null;
}

interface SuperAdminLayoutProps {
  title?: string;
  children: React.ReactNode;
}

export function SuperAdminLayout({ title, children }: SuperAdminLayoutProps) {
  const [user, setUser] = useState<{ name: string; email: string; avatar: string }>({
    name: "Super Admin",
    email: "",
    avatar: "",
  });

  useEffect(() => {
    let cancelled = false;
    apiFetch<Profile>("/auth/profile")
      .then((p) => {
        if (!cancelled) {
          setUser({
            name: p.display_name || p.email || "Super Admin",
            email: p.email || "",
            avatar: p.avatar_url || "",
          });
        }
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <SidebarProvider>
      <SuperAdminSidebar user={user} />
      <SidebarInset>
        <DashboardHeader title={title} />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">
          {children}
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
