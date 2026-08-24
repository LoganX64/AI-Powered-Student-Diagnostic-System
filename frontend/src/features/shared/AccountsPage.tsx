import { useEffect, useState } from "react";
import { toast } from "sonner";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  CircleUserRoundIcon,
  KeyIcon,
  MonitorIcon,
} from "lucide-react";
import { getProfile, type Profile } from "@/services/settings.service";

export function AccountsPage() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const p = await getProfile();
        if (active) setProfile(p);
      } catch (err) {
        toast.error((err as Error).message);
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  return (
    <DashboardLayout title="Accounts">
      <div className="flex flex-col gap-6">
        {/* Account Info */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CircleUserRoundIcon className="size-5" />
              Account Information
            </CardTitle>
            <CardDescription>
              Your account details and status.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {loading || !profile ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : (
              <div className="grid gap-4 md:grid-cols-2 max-w-2xl">
                <div className="flex flex-col gap-1">
                  <Label className="text-muted-foreground">Name</Label>
                  <p className="font-medium">{profile.display_name || "—"}</p>
                </div>
                <div className="flex flex-col gap-1">
                  <Label className="text-muted-foreground">Email</Label>
                  <p className="font-medium">{profile.email}</p>
                </div>
                <div className="flex flex-col gap-1">
                  <Label className="text-muted-foreground">Role</Label>
                  <Badge variant="secondary" className="w-fit capitalize">{profile.role}</Badge>
                </div>
                <div className="flex flex-col gap-1">
                  <Label className="text-muted-foreground">Status</Label>
                  <Badge variant="default" className="w-fit bg-green-500">Active</Badge>
                </div>
                <div className="flex flex-col gap-1">
                  <Label className="text-muted-foreground">Organization</Label>
                  <p className="font-medium">{profile.tenant_name || "—"}</p>
                </div>
                <div className="flex flex-col gap-1">
                  <Label className="text-muted-foreground">Joined</Label>
                  <p className="font-medium">
                    {profile.created_at
                      ? new Date(profile.created_at).toLocaleDateString()
                      : "—"}
                  </p>
                </div>
                <div className="flex flex-col gap-1">
                  <Label className="text-muted-foreground">Phone</Label>
                  <p className="font-medium">{profile.phone || "—"}</p>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* API Keys */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <KeyIcon className="size-5" />
              API Keys
            </CardTitle>
            <CardDescription>
              Manage your API keys for programmatic access.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
              <p className="text-sm text-muted-foreground">Coming Soon</p>
            </div>
          </CardContent>
        </Card>

        {/* Active Sessions */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MonitorIcon className="size-5" />
              Active Sessions
            </CardTitle>
            <CardDescription>
              Manage your active login sessions.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
              <p className="text-sm text-muted-foreground">Coming Soon</p>
            </div>
          </CardContent>
        </Card>
      </div>
    </DashboardLayout>
  );
}
