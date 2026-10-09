import { useCallback, useEffect, useRef, useState } from "react";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
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
import { currentPrefix } from "@/config/routes";
import { toast } from "sonner";

const PAGE_SIZE = 20;

// Severity tabs are the only tabs besides All/Unread, and they map straight onto
// the backend's priority filter. "Alert" was removed: nothing emits that
// priority — only storage_warning can, and it has no producer — so the tab could
// never hold a row. The priority stays valid in the type and in the row
// rendering, so a stored alert would still display correctly under All.
const SEVERITY_TABS = ["info", "warning"] as const;

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
  // The shared store stays pinned to page 1: the header bell needs the newest
  // rows regardless of which page this list is showing. It supplies the
  // offset-independent facts — the org-wide unread count and the viewer's own id.
  const { unreadCount, viewerUserId, refetch: refetchFeed } = useNotifications();

  const [rows, setRows] = useState<Notification[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [activeTab, setActiveTab] = useState("all");
  const [loading, setLoading] = useState(true);

  const reqIdRef = useRef(0);
  const abortRef = useRef<AbortController | null>(null);

  // Filtering happens server-side. Doing it in the browser would only ever see
  // the current page, so a tab could report "no results" while matches sat on
  // page 2 — and the pager's total would describe a different set entirely.
  const fetchPage = useCallback(async (off: number, tab: string) => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    const reqId = ++reqIdRef.current;

    const params = new URLSearchParams({
      limit: String(PAGE_SIZE),
      offset: String(off),
    });
    if (tab === "unread") params.set("unread", "true");
    else if ((SEVERITY_TABS as readonly string[]).includes(tab)) {
      params.set("priority", tab);
    }

    // Only the first load blanks the list. Paging keeps the current rows on
    // screen so the list does not flash empty between pages.
    if (off === 0) setLoading(true);

    try {
      const res = await apiFetch<{
        total: number;
        data: Notification[];
      }>(`${currentPrefix()}/notifications?${params}`, {
        signal: controller.signal,
      });
      // A newer request has started, so this response is stale.
      if (reqId !== reqIdRef.current) return;

      // Deleting the last row of the last page strands the user on an offset
      // past the end. Step back rather than showing a blank page.
      if (off > 0 && (res.data?.length ?? 0) === 0 && off >= res.total) {
        setOffset(Math.max(0, off - PAGE_SIZE));
        return;
      }

      setRows(res.data ?? []);
      setTotal(res.total);
    } catch (err) {
      if ((err as Error)?.name === "AbortError") return;
      toast.error((err as Error).message || "Failed to load notifications");
    } finally {
      if (reqId === reqIdRef.current) setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPage(offset, activeTab);
    return () => abortRef.current?.abort();
  }, [offset, activeTab, fetchPage]);

  // The admin sees every row in the organization, but MarkRead and Delete are
  // owner-scoped server-side, so acting on a coach's notification would 404.
  // Only offer the actions on rows the viewer owns; until the id resolves, show
  // none rather than buttons that cannot work.
  const canActOn = (n: Notification) =>
    viewerUserId !== null && n.user_id === viewerUserId;

  // True when the unread badge counts rows this viewer cannot clear, i.e. the
  // admin looking at org-wide unread. Coaches only ever see their own.
  const ownsEveryVisibleRow =
    viewerUserId !== null && rows.every((n) => n.user_id === viewerUserId);

  // Both sides refresh: this page's rows, and the store's count and viewer id.
  const refreshAll = () => {
    void fetchPage(offset, activeTab);
    void refetchFeed();
  };

  const markAsRead = async (id: number) => {
    const prefix = currentPrefix();
    try {
      await apiFetch(`${prefix}/notifications/${id}/read`, { method: "PUT" });
      refreshAll();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const markAllAsRead = async () => {
    const prefix = currentPrefix();
    try {
      await apiFetch(`${prefix}/notifications/read-all`, { method: "PUT" });
      refreshAll();
      toast.success("All marked as read");
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const deleteNotification = async (id: number) => {
    const prefix = currentPrefix();
    try {
      await apiFetch(`${prefix}/notifications/${id}`, { method: "DELETE" });
      refreshAll();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const handleTabChange = (tab: string) => {
    setActiveTab(tab);
    // Each tab has its own total, so the current offset can point past its end.
    setOffset(0);
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
                  {unreadCount > 0 && !ownsEveryVisibleRow && (
                    <span className="mt-1 block">
                      The unread count covers the whole organization. &ldquo;Mark all
                      as read&rdquo; clears only your own notifications; each
                      user clears their own.
                    </span>
                  )}
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
            <Tabs value={activeTab} onValueChange={handleTabChange}>
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="all">All</TabsTrigger>
                <TabsTrigger value="unread">
                  Unread
                  {unreadCount > 0 && (
                    <Badge variant="destructive" className="ml-1 h-5 px-1.5 text-xs">{unreadCount}</Badge>
                  )}
                </TabsTrigger>
                <TabsTrigger value="info">Info</TabsTrigger>
                <TabsTrigger value="warning">Warning</TabsTrigger>
              </TabsList>

              <TabsContent value={activeTab} className="mt-4">
                {loading && rows.length === 0 ? (
                  <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                    <p className="text-sm text-muted-foreground">Loading…</p>
                  </div>
                ) : rows.length === 0 ? (
                  <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                    <p className="text-sm text-muted-foreground">
                      No notifications in this category.
                    </p>
                  </div>
                ) : (
                  <div className={`flex flex-col gap-2 ${loading ? "opacity-60" : ""}`}>
                    {rows.map((notification) => (
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
                            {!canActOn(notification) && (
                              <span className="ml-2">
                                Read-only — belongs to another user
                              </span>
                            )}
                          </p>
                        </div>
                        {canActOn(notification) && (
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
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </TabsContent>
            </Tabs>

            {/* Pagination. total comes from the server for the active tab, so
                it describes the filtered set rather than every notification. */}
            {total > PAGE_SIZE && (
              <Pagination className="mt-4">
                <PaginationContent className="flex items-center justify-between w-full">
                  <p className="text-sm text-muted-foreground">
                    Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
                  </p>
                  <div className="flex gap-2">
                    <PaginationItem>
                      <PaginationPrevious
                        onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
                        className={offset === 0 ? "pointer-events-none opacity-50" : ""}
                      />
                    </PaginationItem>
                    <PaginationItem>
                      <PaginationNext
                        onClick={() => setOffset((o) => o + PAGE_SIZE)}
                        className={offset + PAGE_SIZE >= total ? "pointer-events-none opacity-50" : ""}
                      />
                    </PaginationItem>
                  </div>
                </PaginationContent>
              </Pagination>
            )}
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  );
}
