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
import { useNotifications } from "@/hooks/useNotifications";
import { useNavigate } from "react-router-dom";
import { getActiveRole, getPrefix } from "@/lib/token";

export function NotificationBell() {
  const role = getActiveRole();
  // No notifications endpoint exists for super_admin — suppress the bell.
  if (role !== "admin" && role !== "coach") return null;

  const { unreadCount, notifications } = useNotifications(30000);
  const navigate = useNavigate();
  const prefix = getPrefix();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="relative">
          <Bell className="size-5" />
          {unreadCount > 0 && (
            <Badge className="absolute -top-1 -right-1 size-5 flex items-center justify-center p-0 text-xs">
              {unreadCount > 99 ? "99+" : unreadCount}
            </Badge>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
        <div className="flex items-center justify-between px-2 py-1.5">
          <span className="text-sm font-semibold">Notifications</span>
          {unreadCount > 0 && (
            <span className="text-xs text-muted-foreground">
              {unreadCount} unread
            </span>
          )}
        </div>
        <DropdownMenuSeparator />
        {notifications.slice(0, 5).map((n) => (
          <DropdownMenuItem
            key={n.id}
            className="flex flex-col items-start gap-1 cursor-pointer"
          >
            <div className="flex items-center gap-2 w-full">
              <span className="text-sm font-medium truncate">{n.title}</span>
              {!n.read_at && (
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
