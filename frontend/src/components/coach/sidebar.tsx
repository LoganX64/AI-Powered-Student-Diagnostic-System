import * as React from "react";
import {
  LayoutDashboardIcon,
  UsersIcon,
  ClipboardListIcon,
  BarChart3Icon,
  MessageSquareIcon,
  Settings2Icon,
  CircleHelpIcon,
  GraduationCapIcon,
  CalendarIcon,
  BookOpenIcon,
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
import { NavMain } from "@/components/admin/nav-main";
import { NavSecondary } from "@/components/admin/nav-secondary";
import { NavUser } from "@/components/admin/nav-user";

const data = {
  user: {
    name: "Coach Alex",
    email: "coach@example.com",
    avatar: "/avatars/coach.jpg",
  },
  navMain: [
    {
      title: "Dashboard",
      url: "/coach/dashboard",
      icon: <LayoutDashboardIcon />,
    },
    {
      title: "My Students",
      url: "#",
      icon: <UsersIcon />,
    },
    {
      title: "Assessments",
      url: "#",
      icon: <ClipboardListIcon />,
    },
    {
      title: "Analytics",
      url: "#",
      icon: <BarChart3Icon />,
    },
    {
      title: "Lessons",
      url: "#",
      icon: <BookOpenIcon />,
    },
    {
      title: "Schedule",
      url: "#",
      icon: <CalendarIcon />,
    },
    {
      title: "Messages",
      url: "#",
      icon: <MessageSquareIcon />,
    },
  ],
  navSecondary: [
    {
      title: "Settings",
      url: "#",
      icon: <Settings2Icon />,
    },
    {
      title: "Get Help",
      url: "#",
      icon: <CircleHelpIcon />,
    },
  ],
};

export function CoachSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              className="data-[slot=sidebar-menu-button]:p-1.5!"
            >
              <a href="/coach/dashboard">
                <GraduationCapIcon className="size-5!" />
                <span className="text-base font-semibold">Coach Portal</span>
              </a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
        <NavSecondary items={data.navSecondary} className="mt-auto" />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={data.user} />
      </SidebarFooter>
    </Sidebar>
  );
}
