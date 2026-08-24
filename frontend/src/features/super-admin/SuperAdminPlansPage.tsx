import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Pencil, Trash2, Plus } from "lucide-react";
import { SuperAdminLayout } from "@/components/super-admin/SuperAdminLayout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { getPlans, createPlan, updatePlan, deletePlan, type Plan, type CreatePlanPayload } from "@/services/super-admin.service";

function formatBytes(bytes: number): string {
  if (bytes === -1) return "Unlimited";
  if (bytes === 0) return "0 GB";
  return `${(bytes / 1073741824).toFixed(0)} GB`;
}

function formatPrice(pricePaise: number): string {
  // ISSUE-6: backend stores price in paise (integer); convert to ₹ for display.
  if (pricePaise === 0) return "Free";
  return `₹${(pricePaise / 100).toLocaleString("en-IN")}/mo`;
}

const emptyForm: CreatePlanPayload = {
  name: "", slug: "", student_limit: 0, coach_limit: 0, storage_limit_bytes: 0,
  test_limit: 0, sqi_access: false, video_proctoring_included: false,
  video_proctoring_limit: 0, video_proctoring_price_per_student: 0, price_monthly: 0, features: [],
};

export function SuperAdminPlansPage() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [editPlan, setEditPlan] = useState<Plan | null>(null);
  const [deletePlanId, setDeletePlanId] = useState<number | null>(null);
  const [form, setForm] = useState<CreatePlanPayload>(emptyForm);
  const [saving, setSaving] = useState(false);

  const fetchPlans = async (shouldAbort?: () => boolean) => {
    try {
      const res = await getPlans();
      if (shouldAbort?.()) return;
      setPlans(res.data ?? []);
    } catch (err) {
      if (!shouldAbort?.()) toast.error((err as Error).message);
    } finally {
      if (!shouldAbort?.()) setLoading(false);
    }
  };

  useEffect(() => {
    let cancelled = false;
    async function load() {
      await fetchPlans(() => cancelled);
    }
    load();
    return () => {
      cancelled = true;
    };
  }, []);

  const openEdit = (plan: Plan) => {
    setEditPlan(plan);
    setForm({
      name: plan.name, slug: plan.slug, student_limit: plan.student_limit, coach_limit: plan.coach_limit,
      storage_limit_bytes: plan.storage_limit_bytes, test_limit: plan.test_limit, sqi_access: plan.sqi_access,
      video_proctoring_included: plan.video_proctoring_included, video_proctoring_limit: plan.video_proctoring_limit,
      // display in ₹ (backend stores paise)
      video_proctoring_price_per_student: plan.video_proctoring_price_per_student / 100,
      price_monthly: plan.price_monthly / 100,
      features: plan.features ?? [],
    });
  };

  const openCreate = () => {
    setEditPlan(null);
    setForm(emptyForm);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const payload: CreatePlanPayload = {
        ...form,
        // convert ₹ -> paise before sending (ISSUE-6)
        price_monthly: Math.round(form.price_monthly * 100),
        video_proctoring_price_per_student: Math.round(form.video_proctoring_price_per_student * 100),
      };
      if (editPlan) {
        await updatePlan(editPlan.id, payload);
        toast.success("Plan updated");
      } else {
        await createPlan(payload);
        toast.success("Plan created");
      }
      setEditPlan(null);
      fetchPlans();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deletePlanId) return;
    try {
      await deletePlan(deletePlanId);
      toast.success("Plan deleted");
      setDeletePlanId(null);
      fetchPlans();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  return (
    <SuperAdminLayout title="Subscription Plans">
      <div className="flex justify-end mb-4">
        <Button onClick={openCreate}>
          <Plus className="size-4 mr-1" /> Create Plan
        </Button>
      </div>

      {loading ? (
        <div className="text-muted-foreground">Loading...</div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {plans.map((plan) => (
            <Card key={plan.id} className="relative">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>{plan.name}</CardTitle>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="icon" className="size-8" onClick={() => openEdit(plan)}>
                      <Pencil className="size-4" />
                    </Button>
                    <Button variant="ghost" size="icon" className="size-8 text-destructive" onClick={() => setDeletePlanId(plan.id)}>
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                </div>
                <CardDescription className="text-2xl font-bold">{formatPrice(plan.price_monthly)}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-2 text-sm">
                <div>Students: {plan.student_limit === -1 ? "Unlimited" : plan.student_limit}</div>
                <div>Coaches: {plan.coach_limit === -1 ? "Unlimited" : plan.coach_limit}</div>
                <div>Storage: {formatBytes(plan.storage_limit_bytes)}</div>
                <div>Tests/mo: {plan.test_limit === -1 ? "Unlimited" : plan.test_limit}</div>
                <div>SQI: {plan.sqi_access ? <Badge>Yes</Badge> : <Badge variant="secondary">No</Badge>}</div>
                <div>Video: {plan.video_proctoring_included ? <Badge>Included ({plan.video_proctoring_limit})</Badge> : <Badge variant="secondary">Add-on</Badge>}</div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={editPlan !== null} onOpenChange={(open) => !open && setEditPlan(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader><DialogTitle>{editPlan?.id ? "Edit Plan" : "Create Plan"}</DialogTitle></DialogHeader>
          <form onSubmit={handleSave} className="flex flex-col gap-4 max-h-[70vh] overflow-y-auto">
            <div className="flex flex-col gap-2"><Label>Name</Label><Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
            <div className="flex flex-col gap-2"><Label>Slug</Label><Input value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} required /></div>
            <div className="flex flex-col gap-2"><Label>Student Limit (-1 = unlimited)</Label><Input type="number" value={form.student_limit} onChange={(e) => setForm({ ...form, student_limit: Number(e.target.value) })} required /></div>
            <div className="flex flex-col gap-2"><Label>Coach Limit (-1 = unlimited)</Label><Input type="number" value={form.coach_limit} onChange={(e) => setForm({ ...form, coach_limit: Number(e.target.value) })} required /></div>
            <div className="flex flex-col gap-2"><Label>Storage Limit (bytes, -1 = unlimited)</Label><Input type="number" value={form.storage_limit_bytes} onChange={(e) => setForm({ ...form, storage_limit_bytes: Number(e.target.value) })} required /></div>
            <div className="flex flex-col gap-2"><Label>Test Limit (-1 = unlimited)</Label><Input type="number" value={form.test_limit} onChange={(e) => setForm({ ...form, test_limit: Number(e.target.value) })} required /></div>
            <div className="flex flex-col gap-2"><Label>Price Monthly (₹)</Label><Input type="number" value={form.price_monthly} onChange={(e) => setForm({ ...form, price_monthly: Number(e.target.value) })} required /></div>
            <div className="flex items-center gap-2"><Switch checked={form.sqi_access} onCheckedChange={(v) => setForm({ ...form, sqi_access: v })} /><Label>SQI Access</Label></div>
            <div className="flex items-center gap-2"><Switch checked={form.video_proctoring_included} onCheckedChange={(v) => setForm({ ...form, video_proctoring_included: v })} /><Label>Video Proctoring Included</Label></div>
            <div className="flex flex-col gap-2"><Label>Video Proctoring Limit</Label><Input type="number" value={form.video_proctoring_limit} onChange={(e) => setForm({ ...form, video_proctoring_limit: Number(e.target.value) })} required /></div>
            <div className="flex flex-col gap-2"><Label>Video Proctoring Price Per Student (₹)</Label><Input type="number" step="0.01" value={form.video_proctoring_price_per_student} onChange={(e) => setForm({ ...form, video_proctoring_price_per_student: Number(e.target.value) })} required /></div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditPlan(null)}>Cancel</Button>
              <Button type="submit" disabled={saving}>{saving ? "Saving..." : "Save"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deletePlanId !== null} onOpenChange={(open) => !open && setDeletePlanId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Plan</AlertDialogTitle>
            <AlertDialogDescription>Are you sure? This cannot be undone.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">Delete</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SuperAdminLayout>
  );
}
