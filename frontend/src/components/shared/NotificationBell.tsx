import { Bell } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { useNotifications, type Notification } from "@/hooks/useNotifications";
import { useNavigate } from "react-router-dom";
import { useRole } from "@/hooks/useRole";
import { prefixForRole } from "@/config/routes";
import { toast } from "sonner";

/**
 * Resolves where a notification click should land, using only metadata the
 * service actually stores. Events without a routable target (coach activity,
 * SQI jobs) return null and the click only marks the row read.
 */
function targetForNotification(n: Notification, prefix: string): string | null {
  const meta = n.metadata ?? {};
  const num = (key: string): number | null => {
    const value = meta[key];
    return typeof value === "number" ? value : null;
  };

  switch (n.event_type) {
    case "exam_submitted":
    case "student_exam_logout": {
      const studentId = num("student_id");
      const assignmentId = num("assignment_id");
      if (studentId === null || assignmentId === null) return null;
      return `${prefix}/students/${studentId}/assignments/${assignmentId}`;
    }
    case "tenant_created":
    case "tenant_suspended":
    case "tenant_reactivated":
    case "plan_changed": {
      const tenantId = num("tenant_id");
      if (tenantId === null) return null;
      return `/super-admin/tenants/${tenantId}`;
    }
    default:
      return null;
  }
}

export function NotificationBell() {
  const role = useRole();
  // Every staff role has endpoints now: admin/coach read tenant rows, a
  // super-admin reads platform rows. The hook must run unconditionally to
  // satisfy rules-of-hooks even when the bell itself is hidden.
  const enabled = role === "admin" || role === "coach" || role === "super_admin";
  const { unseenCount, notifications, markSeen, markRead } = useNotifications(10000, enabled);
  const navigate = useNavigate();
  const prefix = prefixForRole(role);

  if (!enabled) return null;

  const visible = notifications.slice(0, 5);

  // Opening the dropdown is what "seeing" means: clear the badge for the rows on
  // screen, but leave read state alone so the Unread tab still lists them.
  const handleOpenChange = (open: boolean) => {
    if (!open) return;
    const unseenIds = visible.filter((n) => !n.seen_at).map((n) => n.id);
    void markSeen(unseenIds).catch((err) => toast.error((err as Error).message));
  };

  const handleSelect = (n: Notification) => {
    void markRead(n.id).catch((err) => toast.error((err as Error).message));
    const target = targetForNotification(n, prefix);
    if (target) navigate(target);
  };

  return (
    <DropdownMenu onOpenChange={handleOpenChange}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="relative">
          <Bell className="size-5" />
          {unseenCount > 0 && (
            <Badge className="absolute -top-1 -right-1 size-5 flex items-center justify-center p-0 text-xs">
              {unseenCount > 99 ? "99+" : unseenCount}
            </Badge>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
        <div className="flex items-center justify-between px-2 py-1.5">
          <span className="text-sm font-semibold">Notifications</span>
          {unseenCount > 0 && (
            <span className="text-xs text-muted-foreground">
              {unseenCount} new
            </span>
          )}
        </div>
        <DropdownMenuSeparator />
        {visible.map((n) => (
          <DropdownMenuItem
            key={n.id}
            onClick={() => handleSelect(n)}
            className="flex flex-col items-start gap-1 cursor-pointer"
          >
            <div className="flex items-center gap-2 w-full">
              <span
                className={`text-sm truncate ${
                  n.read_at ? "font-medium" : "font-semibold"
                }`}
              >
                {n.title}
              </span>
              {!n.seen_at && (
                <span className="size-2 rounded-full bg-blue-500 shrink-0" />
              )}
            </div>
            <span className="text-xs text-muted-foreground truncate w-full">
              {n.message}
            </span>
          </DropdownMenuItem>
        ))}
        {notifications.length === 0 && (
          <div className="px-2 py-4 text-center text-sm text-muted-foreground">
            No notifications
          </div>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={() => navigate(`${prefix}/notifications`)}
          className="justify-center cursor-pointer"
        >
          View all notifications
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
