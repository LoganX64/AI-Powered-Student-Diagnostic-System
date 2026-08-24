import { useState, useEffect } from "react";
import { toast } from "sonner";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Progress } from "@/components/ui/progress";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { PlanComparison } from "@/components/shared/PlanComparison";
import { bytesToGB } from "@/lib/utils";
import {
  getSubscription,
  getPlans,
  createCheckout,
  cancelSubscription,
  type Subscription,
} from "@/services/billing.service";
import type { Plan } from "@/services/super-admin.service";
import {
  CreditCardIcon,
  ReceiptIcon,
  UsersIcon,
  GraduationCapIcon,
  HardDriveIcon,
  ClipboardListIcon,
  CheckCircle2Icon,
  ArrowUpRightIcon,
} from "lucide-react";

function usagePercent(used: number, limit: number): number {
  if (limit <= 0) return 100; // unlimited or unset -> show full
  return Math.min(100, Math.round((used / limit) * 100));
}

export function BillingPage() {
  const [sub, setSub] = useState<Subscription | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [upgradeOpen, setUpgradeOpen] = useState(false);
  const [cancelOpen, setCancelOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const [subRes, plansRes] = await Promise.all([
          getSubscription(),
          getPlans(),
        ]);
        if (cancelled) return;
        setSub(subRes);
        setPlans(plansRes.data ?? []);
      } catch (err) {
        toast.error((err as Error).message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, []);

  const refresh = async () => {
    try {
      const [subRes, plansRes] = await Promise.all([getSubscription(), getPlans()]);
      setSub(subRes);
      setPlans(plansRes.data ?? []);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const handleUpgrade = async (slug: string) => {
    setSaving(true);
    try {
      await createCheckout(slug);
      toast.success("Plan updated");
      setUpgradeOpen(false);
      await refresh();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleCancel = async () => {
    setSaving(true);
    try {
      await cancelSubscription();
      toast.success("Subscription cancelled");
      setCancelOpen(false);
      await refresh();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const currentPlan = plans.find((p) => p.id === sub?.plan_id);

  return (
    <DashboardLayout title="Billing">
      <div className="flex flex-col gap-6">
        {/* Current Plan */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CreditCardIcon className="size-5" />
              Current Plan
            </CardTitle>
            <CardDescription>
              Your subscription details and billing information.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {loading || !sub ? (
              <p className="text-sm text-muted-foreground">Loading plan...</p>
            ) : (
              <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                <div className="flex flex-col gap-2">
                  <div className="flex items-center gap-2">
                    <h3 className="text-2xl font-bold">{sub.plan_name}</h3>
                    <Badge>{sub.status}</Badge>
                  </div>
                  <p className="text-3xl font-bold">
                    ₹{(sub.additional_cost_paise / 100).toLocaleString("en-IN")}
                    <span className="text-sm font-normal text-muted-foreground">
                      /mo
                    </span>
                  </p>
                  {sub.additional_cost_paise > 0 && (
                    <p className="text-sm text-amber-600">
                      Includes ₹{(sub.additional_cost_paise / 100).toLocaleString("en-IN")} storage overage
                    </p>
                  )}
                  <p className="text-sm text-muted-foreground">
                    {sub.current_period_start && sub.current_period_end
                      ? `Billing period: ${new Date(sub.current_period_start).toLocaleDateString()} – ${new Date(sub.current_period_end).toLocaleDateString()}`
                      : "Billing period: —"}
                  </p>
                </div>
                <div className="flex flex-col gap-2">
                  <Button className="w-fit" onClick={() => setUpgradeOpen(true)}>
                    Upgrade Plan
                    <ArrowUpRightIcon className="size-4 ml-1" />
                  </Button>
                  <Button variant="outline" className="w-fit" onClick={() => setCancelOpen(true)}>
                    Cancel Subscription
                  </Button>
                </div>
              </div>
            )}
            <Separator className="my-4" />
            <div>
              <h4 className="font-medium mb-2">Plan Features</h4>
              {currentPlan?.features?.length ? (
                <ul className="grid gap-2 md:grid-cols-2">
                  {currentPlan.features.map((feature) => (
                    <li key={feature} className="flex items-center gap-2 text-sm">
                      <CheckCircle2Icon className="size-4 text-green-500" />
                      {feature}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-sm text-muted-foreground">No feature details available.</p>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Usage */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <HardDriveIcon className="size-5" />
              Usage
            </CardTitle>
            <CardDescription>
              Your current resource usage against plan limits.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {loading || !sub ? (
              <p className="text-sm text-muted-foreground">Loading usage...</p>
            ) : (
              <div className="grid gap-6 md:grid-cols-2">
                <div className="flex flex-col gap-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <GraduationCapIcon className="size-4 text-muted-foreground" />
                      <span className="text-sm font-medium">Students</span>
                    </div>
                    <span className="text-sm text-muted-foreground">
                      {sub.student_count} / {sub.student_limit === -1 ? "Unlimited" : sub.student_limit}
                    </span>
                  </div>
                  <Progress value={usagePercent(sub.student_count, sub.student_limit)} className="h-2" />
                </div>
                <div className="flex flex-col gap-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <UsersIcon className="size-4 text-muted-foreground" />
                      <span className="text-sm font-medium">Coaches</span>
                    </div>
                    <span className="text-sm text-muted-foreground">
                      {sub.coach_count} / {sub.coach_limit === -1 ? "Unlimited" : sub.coach_limit}
                    </span>
                  </div>
                  <Progress value={usagePercent(sub.coach_count, sub.coach_limit)} className="h-2" />
                </div>
                <div className="flex flex-col gap-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <HardDriveIcon className="size-4 text-muted-foreground" />
                      <span className="text-sm font-medium">Storage</span>
                    </div>
                    <span className="text-sm text-muted-foreground">
                      {bytesToGB(sub.storage_used_bytes)} / {bytesToGB(sub.storage_limit_bytes)}
                    </span>
                  </div>
                  <Progress value={usagePercent(Number(sub.storage_used_bytes), Number(sub.storage_limit_bytes))} className="h-2" />
                </div>
                <div className="flex flex-col gap-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <ClipboardListIcon className="size-4 text-muted-foreground" />
                      <span className="text-sm font-medium">Tests (this month)</span>
                    </div>
                    <span className="text-sm text-muted-foreground">
                      {sub.test_count_this_month} / {sub.test_limit === -1 ? "Unlimited" : sub.test_limit}
                    </span>
                  </div>
                  <Progress value={usagePercent(sub.test_count_this_month, sub.test_limit)} className="h-2" />
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Billing History */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ReceiptIcon className="size-5" />
              Billing History
            </CardTitle>
            <CardDescription>
              Your past invoices and payment records.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">Coming soon.</p>
          </CardContent>
        </Card>
      </div>

      {/* Upgrade dialog */}
      <Dialog open={upgradeOpen} onOpenChange={setUpgradeOpen}>
        <DialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>Compare Plans</DialogTitle>
          </DialogHeader>
          {sub && (
            <PlanComparison
              plans={plans}
              currentPlanId={sub.plan_id}
              onSelectPlan={handleUpgrade}
            />
          )}
          {saving && <p className="text-sm text-muted-foreground">Updating plan...</p>}
        </DialogContent>
      </Dialog>

      {/* Cancel confirmation */}
      <AlertDialog open={cancelOpen} onOpenChange={setCancelOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cancel Subscription</AlertDialogTitle>
            <AlertDialogDescription>
              This will cancel your current plan. You will be moved to the Free plan. Are you sure?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep Subscription</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleCancel}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {saving ? "Cancelling..." : "Cancel Subscription"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </DashboardLayout>
  );
}
