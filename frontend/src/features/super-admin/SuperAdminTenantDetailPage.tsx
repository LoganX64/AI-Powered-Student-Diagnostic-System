import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { ArrowLeft, Plus, CreditCard } from "lucide-react";
import { SuperAdminLayout } from "@/components/super-admin/SuperAdminLayout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { getTenant, getTenantAdmins, getPlans, createTenantAdmin, assignPlan, type Tenant, type User, type Plan } from "@/services/super-admin.service";

export function SuperAdminTenantDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [admins, setAdmins] = useState<User[]>([]);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [addAdminOpen, setAddAdminOpen] = useState(false);
  const [assignOpen, setAssignOpen] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState<string>("");
  const [adminForm, setAdminForm] = useState({ email: "", password: "", name: "" });
  const [saving, setSaving] = useState(false);

  const fetchAll = useCallback(async (shouldAbort?: () => boolean) => {
    if (!id) return;
    setLoading(true);
    try {
      const [t, a, p] = await Promise.all([
        getTenant(Number(id)),
        getTenantAdmins(Number(id)),
        getPlans(),
      ]);
      if (shouldAbort?.()) return;
      setTenant(t);
      setAdmins(a.data ?? []);
      setPlans(p.data ?? []);
      setSelectedPlan(t.plan_id ? String(t.plan_id) : "");
    } catch (err) {
      if (!shouldAbort?.()) toast.error((err as Error).message);
    } finally {
      if (!shouldAbort?.()) setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      await fetchAll(() => cancelled);
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [fetchAll]);

  if (loading) return <SuperAdminLayout title="Tenant Details"><div>Loading...</div></SuperAdminLayout>;
  if (!tenant) return <SuperAdminLayout title="Tenant Details"><div>Tenant not found</div></SuperAdminLayout>;

  const handleAddAdmin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id) return;
    setSaving(true);
    try {
      await createTenantAdmin(Number(id), adminForm);
      toast.success("Admin created");
      setAddAdminOpen(false);
      fetchAll();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleAssign = async () => {
    if (!id || !selectedPlan) return;
    setSaving(true);
    try {
      await assignPlan(Number(id), Number(selectedPlan));
      toast.success("Plan assigned");
      setAssignOpen(false);
      fetchAll();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const currentPlan = plans.find((p) => String(p.id) === String(tenant.plan_id));

  return (
    <SuperAdminLayout title={`Tenant: ${tenant.name}`}>
      <Button variant="ghost" onClick={() => navigate("/super-admin/tenants")} className="w-fit">
        <ArrowLeft className="size-4 mr-1" /> Back to Tenants
      </Button>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader><CardTitle className="text-sm">Students</CardTitle></CardHeader>
          <CardContent><div className="text-2xl font-bold">{tenant.student_count}</div></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm">Coaches</CardTitle></CardHeader>
          <CardContent><div className="text-2xl font-bold">{tenant.coach_count}</div></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-sm">Users</CardTitle></CardHeader>
          <CardContent><div className="text-2xl font-bold">{tenant.user_count}</div></CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>
            Plan: {currentPlan ? currentPlan.name : "—"}
            {tenant.suspended_at && <Badge variant="destructive" className="ml-2">Suspended</Badge>}
          </CardTitle>
          <Button size="sm" variant="outline" onClick={() => setAssignOpen(true)}>
            <CreditCard className="size-3 mr-1" /> Assign Plan
          </Button>
        </CardHeader>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Admins</CardTitle>
          <Button size="sm" onClick={() => setAddAdminOpen(true)}>
            <Plus className="size-3 mr-1" /> Add Admin
          </Button>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Created At</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {admins.length === 0 ? (
                  <TableRow><TableCell colSpan={4} className="text-center py-8 text-muted-foreground">No admins</TableCell></TableRow>
                ) : admins.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell className="font-mono text-sm">{a.id}</TableCell>
                    <TableCell>{a.email}</TableCell>
                    <TableCell><Badge>{a.role}</Badge></TableCell>
                    <TableCell>{a.created_at}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <Dialog open={addAdminOpen} onOpenChange={setAddAdminOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>Add Admin</DialogTitle></DialogHeader>
          <form onSubmit={handleAddAdmin} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label>Name</Label>
              <Input value={adminForm.name} onChange={(e) => setAdminForm({ ...adminForm, name: e.target.value })} required />
            </div>
            <div className="flex flex-col gap-2">
              <Label>Email</Label>
              <Input type="email" value={adminForm.email} onChange={(e) => setAdminForm({ ...adminForm, email: e.target.value })} required />
            </div>
            <div className="flex flex-col gap-2">
              <Label>Password</Label>
              <Input type="password" value={adminForm.password} onChange={(e) => setAdminForm({ ...adminForm, password: e.target.value })} required />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAddAdminOpen(false)}>Cancel</Button>
              <Button type="submit" disabled={saving}>{saving ? "Creating..." : "Create"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={assignOpen} onOpenChange={setAssignOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>Assign Plan</DialogTitle></DialogHeader>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label>Subscription Plan</Label>
              <Select value={selectedPlan} onValueChange={setSelectedPlan}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a plan" />
                </SelectTrigger>
                <SelectContent>
                  {plans.map((p) => (
                    <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAssignOpen(false)}>Cancel</Button>
              <Button type="button" disabled={saving || !selectedPlan} onClick={handleAssign}>{saving ? "Saving..." : "Assign"}</Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </SuperAdminLayout>
  );
}
