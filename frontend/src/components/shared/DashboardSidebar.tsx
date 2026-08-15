import * as React from "react";
import {
  LayoutDashboardIcon,
  UsersIcon,
  GraduationCapIcon,
  BookOpenIcon,
  ClipboardListIcon,
  FilePlusIcon,
  FolderIcon,
  Settings2Icon,
  CircleHelpIcon,
  CommandIcon,
} from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { NavMain } from "@/components/shared/nav-main";
import { NavSecondary } from "@/components/shared/nav-secondary";
import { NavUser } from "@/components/shared/nav-user";
import { useRole } from "@/hooks/useRole";

const adminNavItems = [
  {
    title: "Dashboard",
    url: "/admin/dashboard",
    icon: <LayoutDashboardIcon />,
  },
  {
    title: "Coaches",
    url: "/admin/coaches",
    icon: <UsersIcon />,
  },
  {
    title: "Students",
    url: "/admin/students",
    icon: <GraduationCapIcon />,
  },
  {
    title: "Batches",
    url: "/admin/batches",
    icon: <FolderIcon />,
  },
  {
    title: "Subjects",
    url: "/admin/subjects",
    icon: <BookOpenIcon />,
  },
  {
    title: "Create Test",
    url: "/admin/tests",
    icon: <FilePlusIcon />,
  },
  {
    title: "All Tests",
    url: "/admin/all-tests",
    icon: <ClipboardListIcon />,
  },
];

const coachNavItems = [
  {
    title: "Dashboard",
    url: "/coach/dashboard",
    icon: <LayoutDashboardIcon />,
  },
  {
    title: "Students",
    url: "/coach/students",
    icon: <GraduationCapIcon />,
  },
  {
    title: "Batches",
    url: "/coach/batches",
    icon: <FolderIcon />,
  },
  {
    title: "Subjects",
    url: "/coach/subjects",
    icon: <BookOpenIcon />,
  },
  {
    title: "Create Test",
    url: "/coach/tests",
    icon: <FilePlusIcon />,
  },
  {
    title: "All Tests",
    url: "/coach/all-tests",
    icon: <ClipboardListIcon />,
  },
];

export function DashboardSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";

  const secondaryNavItems = [
    {
      title: "Settings",
      url: `${prefix}/settings`,
      icon: <Settings2Icon />,
    },
    {
      title: "Get Help",
      url: `${prefix}/help`,
      icon: <CircleHelpIcon />,
    },
  ];

  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              className="data-[slot=sidebar-menu-button]:p-1.5!"
            >
              <a href={`${prefix}/dashboard`}>
                <CommandIcon className="size-5!" />
                <span className="text-base font-semibold">
                  {role === "admin" ? "Admin Panel" : "Coach Portal"}
                </span>
              </a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={role === "admin" ? adminNavItems : coachNavItems} />
        <NavSecondary items={secondaryNavItems} className="mt-auto" />
      </SidebarContent>
      <SidebarFooter>
        <NavUser
          user={{
            name: role === "admin" ? "Admin User" : "Coach Alex",
            email: role === "admin" ? "admin@example.com" : "coach@example.com",
            avatar: role === "admin" ? "/avatars/shadcn.jpg" : "/avatars/coach.jpg",
          }}
        />
      </SidebarFooter>
    </Sidebar>
  );
}
