import { BellOffIcon } from "lucide-react";
import { SuperAdminLayout } from "@/components/super-admin/SuperAdminLayout";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

/**
 * Placeholder for /super-admin/notifications.
 *
 * The notifications table is tenant-scoped and cannot hold a super-admin row:
 * `notifications.tenant_id` is NOT NULL with a foreign key to tenants, while a
 * super-admin user has no tenant (users.tenant_id is nullable). Every query in
 * notification_repo.go filters on `tenant_id = $1` as the primary key of the
 * result set, so there is no valid tenant_id a super-admin could read. Wiring
 * this page to the existing endpoint would 403 on every call.
 *
 * Until a platform-scope notification model exists, say so plainly rather than
 * firing a request that cannot succeed.
 */
export function SuperAdminNotificationsPage() {
  return (
    <SuperAdminLayout title="Notifications">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BellOffIcon className="size-5" />
            Notifications are not available here
          </CardTitle>
          <CardDescription>
            Notifications are scoped to a single organization, and the super-admin
            panel sits above all of them rather than inside one.
          </CardDescription>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          <p>
            Tenant activity that affects everyone — new signups, suspensions, plan
            changes, quota breaches — is not surfaced yet. See each tenant from the
            Tenants page for its current state.
          </p>
        </CardContent>
      </Card>
    </SuperAdminLayout>
  );
}