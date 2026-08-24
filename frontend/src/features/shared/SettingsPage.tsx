import { useState, useEffect } from "react";
import { toast } from "sonner";
import { profileSettingsSchema, changePasswordSchema, zodErrors } from "@/lib/validations";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import {
  UserIcon,
  BellIcon,
  ShieldIcon,
  PaletteIcon,
  Building2Icon,
} from "lucide-react";
import {
  getProfile,
  updateProfile,
  updatePassword,
  updateTenantName,
  getNotificationPreferences,
  updateNotificationPreferences,
  type Profile,
} from "@/services/settings.service";
import { useRole } from "@/hooks/useRole";

const NOTIFICATION_EVENTS: { event_type: string; label: string; description: string }[] = [
  { event_type: "exam_submitted", label: "Exam Submitted", description: "When a student submits an exam" },
  { event_type: "coach_activity", label: "Coach Activity", description: "Test and assignment activity by coaches" },
  { event_type: "system_alert", label: "System Alerts", description: "Warnings about storage and system health" },
  { event_type: "sqi_complete", label: "SQI Complete", description: "When an SQI computation job finishes" },
  { event_type: "storage_warning", label: "Storage Warning", description: "Storage quota warning and critical alerts" },
  { event_type: "student_exam_logout", label: "Student Exam Logout", description: "When a student logs out during an active exam" },
];

export function SettingsPage() {
  const role = useRole();
  const isAdmin = role === "admin";

  const [profile, setProfile] = useState<Profile | null>(null);
  const [displayName, setDisplayName] = useState("");
  const [phone, setPhone] = useState("");
  const [tenantName, setTenantName] = useState("");

  const [prefs, setPrefs] = useState<Record<string, boolean>>({});
  const [prefsLoaded, setPrefsLoaded] = useState(false);

  const [notificationsLoading, setNotificationsLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [profileErrors, setProfileErrors] = useState<Record<string, string>>({});
  const [passwordErrors, setPasswordErrors] = useState<Record<string, string>>({});
  const [passwords, setPasswords] = useState({
    currentPassword: "",
    newPassword: "",
    confirmNewPassword: "",
  });

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const p = await getProfile();
        if (cancelled) return;
        setProfile(p);
        setDisplayName(p.display_name ?? "");
        setPhone(p.phone ?? "");
        setTenantName(p.tenant_name ?? "");
      } catch (err) {
        toast.error((err as Error).message);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    async function loadPrefs() {
      setNotificationsLoading(true);
      try {
        const res = await getNotificationPreferences();
        if (cancelled) return;
        const map: Record<string, boolean> = {};
        res.preferences.forEach((pref) => {
          map[pref.event_type] = pref.enabled;
        });
        setPrefs(map);
        setPrefsLoaded(true);
      } catch (err) {
        toast.error((err as Error).message);
      } finally {
        if (!cancelled) setNotificationsLoading(false);
      }
    }
    loadPrefs();
    return () => {
      cancelled = true;
    };
  }, []);

  const handleSaveProfile = async (e: React.FormEvent) => {
    e.preventDefault();

    const result = profileSettingsSchema.safeParse({
      name: displayName,
      email: profile?.email ?? "",
      phone,
    });
    if (!result.success) {
      setProfileErrors(zodErrors(result.error));
      return;
    }
    setProfileErrors({});

    setSaving(true);
    try {
      await updateProfile({ display_name: displayName, phone });
      toast.success("Profile updated successfully");
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleSaveNotifications = async () => {
    setSaving(true);
    try {
      await updateNotificationPreferences(prefs);
      toast.success("Notification preferences saved");
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleSavePassword = async () => {
    const result = changePasswordSchema.safeParse(passwords);
    if (!result.success) {
      setPasswordErrors(zodErrors(result.error));
      return;
    }
    setPasswordErrors({});

    setSaving(true);
    try {
      await updatePassword({
        current_password: passwords.currentPassword,
        new_password: passwords.newPassword,
      });
      toast.success("Password updated successfully");
      setPasswords({ currentPassword: "", newPassword: "", confirmNewPassword: "" });
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleSaveTenant = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tenantName.trim()) {
      toast.error("Organization name is required");
      return;
    }
    setSaving(true);
    try {
      await updateTenantName(tenantName.trim());
      toast.success("Organization name updated");
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <DashboardLayout title="Settings">
      <Tabs defaultValue="profile" className="w-full">
        <TabsList className={isAdmin ? "grid w-full grid-cols-5" : "grid w-full grid-cols-4"}>
          <TabsTrigger value="profile" className="flex items-center gap-2">
            <UserIcon className="size-4" />
            Profile
          </TabsTrigger>
          <TabsTrigger value="notifications" className="flex items-center gap-2">
            <BellIcon className="size-4" />
            Notifications
          </TabsTrigger>
          {isAdmin && (
            <TabsTrigger value="tenant" className="flex items-center gap-2">
              <Building2Icon className="size-4" />
              Organization
            </TabsTrigger>
          )}
          <TabsTrigger value="appearance" className="flex items-center gap-2">
            <PaletteIcon className="size-4" />
            Appearance
          </TabsTrigger>
          <TabsTrigger value="security" className="flex items-center gap-2">
            <ShieldIcon className="size-4" />
            Security
          </TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <UserIcon className="size-5" />
                Profile Settings
              </CardTitle>
              <CardDescription>
                Manage your personal information and preferences.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSaveProfile} className="flex flex-col gap-4 max-w-lg">
                <div className="flex flex-col gap-2">
                  <Label htmlFor="name">Full Name</Label>
                  <Input
                    id="name"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                  />
                  {profileErrors.name && <p className="text-sm text-destructive">{profileErrors.name}</p>}
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    value={profile?.email ?? ""}
                    disabled
                    onChange={() => {}}
                  />
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="phone">Phone</Label>
                  <Input
                    id="phone"
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                  />
                  {profileErrors.phone && <p className="text-sm text-destructive">{profileErrors.phone}</p>}
                </div>
                <div className="flex items-center gap-2">
                  {profile?.role && <Badge variant="secondary">{profile.role}</Badge>}
                  {profile?.created_at && (
                    <span className="text-sm text-muted-foreground">
                      Member since {new Date(profile.created_at).toLocaleDateString()}
                    </span>
                  )}
                </div>
                <Button type="submit" disabled={saving} className="w-fit">
                  {saving ? "Saving..." : "Save Changes"}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="notifications" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <BellIcon className="size-5" />
                Notification Preferences
              </CardTitle>
              <CardDescription>
                Choose which events you want to be notified about.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-6 max-w-lg">
              {notificationsLoading && (
                <p className="text-sm text-muted-foreground">Loading preferences...</p>
              )}
              {prefsLoaded &&
                NOTIFICATION_EVENTS.map((evt, idx) => (
                  <div key={evt.event_type}>
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>{evt.label}</Label>
                        <p className="text-sm text-muted-foreground">{evt.description}</p>
                      </div>
                      <Switch
                        checked={prefs[evt.event_type] ?? false}
                        onCheckedChange={(checked) =>
                          setPrefs((prev) => ({ ...prev, [evt.event_type]: checked }))
                        }
                      />
                    </div>
                    {idx < NOTIFICATION_EVENTS.length - 1 && <Separator className="mt-6" />}
                  </div>
                ))}
              <Button onClick={handleSaveNotifications} disabled={saving || !prefsLoaded} className="w-fit">
                {saving ? "Saving..." : "Save Preferences"}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>

        {isAdmin && (
          <TabsContent value="tenant" className="mt-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Building2Icon className="size-5" />
                  Organization Settings
                </CardTitle>
                <CardDescription>
                  Manage your organization name and details.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleSaveTenant} className="flex flex-col gap-4 max-w-lg">
                  <div className="flex flex-col gap-2">
                    <Label htmlFor="tenant-name">Organization Name</Label>
                    <Input
                      id="tenant-name"
                      value={tenantName}
                      onChange={(e) => setTenantName(e.target.value)}
                    />
                  </div>
                  <Button type="submit" disabled={saving} className="w-fit">
                    {saving ? "Saving..." : "Save Changes"}
                  </Button>
                </form>
              </CardContent>
            </Card>
          </TabsContent>
        )}

        <TabsContent value="appearance" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <PaletteIcon className="size-5" />
                Appearance
              </CardTitle>
              <CardDescription>
                Customize the look and feel of your dashboard.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                This app uses a light theme.
              </p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShieldIcon className="size-5" />
                Security
              </CardTitle>
              <CardDescription>
                Manage your password and security settings.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4 max-w-lg">
              <div className="flex flex-col gap-2">
                <Label htmlFor="current-password">Current Password</Label>
                <Input
                  id="current-password"
                  type="password"
                  value={passwords.currentPassword}
                  onChange={(e) => setPasswords({ ...passwords, currentPassword: e.target.value })}
                />
                {passwordErrors.currentPassword && <p className="text-sm text-destructive">{passwordErrors.currentPassword}</p>}
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="new-password">New Password</Label>
                <Input
                  id="new-password"
                  type="password"
                  value={passwords.newPassword}
                  onChange={(e) => setPasswords({ ...passwords, newPassword: e.target.value })}
                />
                {passwordErrors.newPassword && <p className="text-sm text-destructive">{passwordErrors.newPassword}</p>}
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="confirm-password">Confirm New Password</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  value={passwords.confirmNewPassword}
                  onChange={(e) => setPasswords({ ...passwords, confirmNewPassword: e.target.value })}
                />
                {passwordErrors.confirmNewPassword && <p className="text-sm text-destructive">{passwordErrors.confirmNewPassword}</p>}
              </div>
              <Button className="w-fit" onClick={handleSavePassword} disabled={saving}>
                {saving ? "Updating..." : "Update Password"}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </DashboardLayout>
  );
}
