import { useState } from "react";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  BellIcon,
  AlertTriangleIcon,
  InfoIcon,
  AlertCircleIcon,
  CheckIcon,
  Trash2Icon,
} from "lucide-react";
import { useNotifications, type Notification } from "@/hooks/useNotifications";
import { apiFetch } from "@/lib/api";
import { getPrefix } from "@/lib/token";
import { toast } from "sonner";

function formatEventType(eventType: string): string {
  return eventType
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}

function getPriorityIcon(priority: Notification["priority"]) {
  switch (priority) {
    case "info":
      return <InfoIcon className="size-4 text-blue-500" />;
    case "warning":
      return <AlertTriangleIcon className="size-4 text-yellow-500" />;
    case "alert":
      return <AlertCircleIcon className="size-4 text-red-500" />;
  }
}

function getPriorityBadge(priority: Notification["priority"]) {
  switch (priority) {
    case "info":
      return <Badge variant="secondary">Info</Badge>;
    case "warning":
      return <Badge variant="outline" className="border-yellow-500 text-yellow-600">Warning</Badge>;
    case "alert":
      return <Badge variant="destructive">Alert</Badge>;
  }
}

export function NotificationsPage() {
  const { notifications, unreadCount, loading, refetch } = useNotifications(30000);
  const [activeTab, setActiveTab] = useState("all");

  const filtered = notifications.filter((n) => {
    if (activeTab === "all") return true;
    if (activeTab === "unread") return !n.read_at;
    return n.priority === activeTab;
  });

  const markAsRead = async (id: number) => {
    const prefix = getPrefix();
    try {
      await apiFetch(`${prefix}/notifications/${id}/read`, { method: "PUT" });
      refetch();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const markAllAsRead = async () => {
    const prefix = getPrefix();
    try {
      await apiFetch(`${prefix}/notifications/read-all`, { method: "PUT" });
      refetch();
      toast.success("All marked as read");
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const deleteNotification = async (id: number) => {
    const prefix = getPrefix();
    try {
      await apiFetch(`${prefix}/notifications/${id}`, { method: "DELETE" });
      refetch();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  return (
    <DashboardLayout title="Notifications">
      <div className="flex flex-col gap-6">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <BellIcon className="size-5" />
                  Notifications
                  {unreadCount > 0 && (
                    <Badge variant="destructive" className="ml-2">{unreadCount}</Badge>
                  )}
                </CardTitle>
                <CardDescription>
                  Stay updated with the latest activity and alerts.
                </CardDescription>
              </div>
              {unreadCount > 0 && (
                <Button variant="outline" onClick={markAllAsRead}>
                  <CheckIcon className="size-4 mr-1" />
                  Mark all as read
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <TabsList className="grid w-full grid-cols-5">
                <TabsTrigger value="all">All</TabsTrigger>
                <TabsTrigger value="unread">
                  Unread
                  {unreadCount > 0 && (
                    <Badge variant="destructive" className="ml-1 h-5 px-1.5 text-xs">{unreadCount}</Badge>
                  )}
                </TabsTrigger>
                <TabsTrigger value="info">Info</TabsTrigger>
                <TabsTrigger value="warning">Warning</TabsTrigger>
                <TabsTrigger value="alert">Alert</TabsTrigger>
              </TabsList>

              <TabsContent value={activeTab} className="mt-4">
                {loading ? (
                  <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                    <p className="text-sm text-muted-foreground">Loading…</p>
                  </div>
                ) : filtered.length === 0 ? (
                  <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                    <p className="text-sm text-muted-foreground">
                      No notifications in this category.
                    </p>
                  </div>
                ) : (
                  <div className="flex flex-col gap-2">
                    {filtered.map((notification) => (
                      <div
                        key={notification.id}
                        className={`flex items-start gap-3 p-4 rounded-lg border transition-colors ${
                          !notification.read_at
                            ? "bg-accent/50 border-primary/20"
                            : "hover:bg-accent/30"
                        }`}
                      >
                        <div className="mt-0.5">{getPriorityIcon(notification.priority)}</div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <h4 className="font-medium text-sm">{notification.title}</h4>
                            {getPriorityBadge(notification.priority)}
                            <Badge variant="outline" className="text-xs">
                              {formatEventType(notification.event_type)}
                            </Badge>
                            {!notification.read_at && (
                              <div className="h-2 w-2 rounded-full bg-primary" />
                            )}
                          </div>
                          <p className="text-sm text-muted-foreground">{notification.message}</p>
                          <p className="text-xs text-muted-foreground mt-1">
                            {new Date(notification.created_at).toLocaleString()}
                          </p>
                        </div>
                        <div className="flex gap-1">
                          {!notification.read_at && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className="size-8"
                              onClick={() => markAsRead(notification.id)}
                            >
                              <CheckIcon className="size-4" />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="icon"
                            className="size-8 text-destructive"
                            onClick={() => deleteNotification(notification.id)}
                          >
                            <Trash2Icon className="size-4" />
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  );
}
