import { useState } from "react";
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
} from "lucide-react";
import {
  mockUserProfile,
  mockNotificationPreferences,
} from "@/features/shared/mockData";

export function SettingsPage() {
  const [profile, setProfile] = useState(mockUserProfile);
  const [notifications, setNotifications] = useState(mockNotificationPreferences);
  const [saving, setSaving] = useState(false);
  const [profileErrors, setProfileErrors] = useState<Record<string, string>>({});
  const [passwordErrors, setPasswordErrors] = useState<Record<string, string>>({});
  const [passwords, setPasswords] = useState({
    currentPassword: "",
    newPassword: "",
    confirmNewPassword: "",
  });

  const handleSaveProfile = async (e: React.FormEvent) => {
    e.preventDefault();

    const result = profileSettingsSchema.safeParse(profile);
    if (!result.success) {
      setProfileErrors(zodErrors(result.error));
      return;
    }
    setProfileErrors({});

    setSaving(true);
    await new Promise((r) => setTimeout(r, 1000));
    toast.success("Profile updated successfully");
    setSaving(false);
  };

  const handleSaveNotifications = async () => {
    setSaving(true);
    await new Promise((r) => setTimeout(r, 1000));
    toast.success("Notification preferences saved");
    setSaving(false);
  };

  const handleSavePassword = async () => {
    const result = changePasswordSchema.safeParse(passwords);
    if (!result.success) {
      setPasswordErrors(zodErrors(result.error));
      return;
    }
    setPasswordErrors({});

    setSaving(true);
    await new Promise((r) => setTimeout(r, 1000));
    toast.success("Password updated successfully");
    setPasswords({ currentPassword: "", newPassword: "", confirmNewPassword: "" });
    setSaving(false);
  };

  return (
    <DashboardLayout title="Settings">
    <Tabs defaultValue="profile" className="w-full">
      <TabsList className="grid w-full grid-cols-4">
        <TabsTrigger value="profile" className="flex items-center gap-2">
          <UserIcon className="size-4" />
          Profile
        </TabsTrigger>
        <TabsTrigger value="notifications" className="flex items-center gap-2">
          <BellIcon className="size-4" />
          Notifications
        </TabsTrigger>
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
                  value={profile.name}
                  onChange={(e) => setProfile({ ...profile, name: e.target.value })}
                />
                {profileErrors.name && <p className="text-sm text-destructive">{profileErrors.name}</p>}
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={profile.email}
                  onChange={(e) => setProfile({ ...profile, email: e.target.value })}
                />
                {profileErrors.email && <p className="text-sm text-destructive">{profileErrors.email}</p>}
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="phone">Phone</Label>
                <Input
                  id="phone"
                  value={profile.phone}
                  onChange={(e) => setProfile({ ...profile, phone: e.target.value })}
                />
                {profileErrors.phone && <p className="text-sm text-destructive">{profileErrors.phone}</p>}
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="secondary">{profile.role}</Badge>
                <span className="text-sm text-muted-foreground">
                  Member since {new Date(profile.joinedAt).toLocaleDateString()}
                </span>
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
              Choose how you want to be notified about updates.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-6 max-w-lg">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Email Notifications</Label>
                <p className="text-sm text-muted-foreground">
                  Receive notifications via email
                </p>
              </div>
              <Switch
                checked={notifications.emailNotifications}
                onCheckedChange={(checked) =>
                  setNotifications({ ...notifications, emailNotifications: checked })
                }
              />
            </div>
            <Separator />
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Push Notifications</Label>
                <p className="text-sm text-muted-foreground">
                  Receive push notifications in browser
                </p>
              </div>
              <Switch
                checked={notifications.pushNotifications}
                onCheckedChange={(checked) =>
                  setNotifications({ ...notifications, pushNotifications: checked })
                }
              />
            </div>
            <Separator />
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Weekly Digest</Label>
                <p className="text-sm text-muted-foreground">
                  Get a weekly summary of activity
                </p>
              </div>
              <Switch
                checked={notifications.weeklyDigest}
                onCheckedChange={(checked) =>
                  setNotifications({ ...notifications, weeklyDigest: checked })
                }
              />
            </div>
            <Separator />
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Test Alerts</Label>
                <p className="text-sm text-muted-foreground">
                  Alerts for test submissions and deadlines
                </p>
              </div>
              <Switch
                checked={notifications.testAlerts}
                onCheckedChange={(checked) =>
                  setNotifications({ ...notifications, testAlerts: checked })
                }
              />
            </div>
            <Separator />
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Student Activity</Label>
                <p className="text-sm text-muted-foreground">
                  Notifications about student progress
                </p>
              </div>
              <Switch
                checked={notifications.studentActivity}
                onCheckedChange={(checked) =>
                  setNotifications({ ...notifications, studentActivity: checked })
                }
              />
            </div>
            <Button onClick={handleSaveNotifications} disabled={saving} className="w-fit">
              {saving ? "Saving..." : "Save Preferences"}
            </Button>
          </CardContent>
        </Card>
      </TabsContent>

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
              Theme customization coming soon. Currently using the default theme.
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
