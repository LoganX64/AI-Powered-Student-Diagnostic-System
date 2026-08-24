import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Plus, Search, Pencil, Ban, RotateCcw } from "lucide-react";
import { SuperAdminLayout } from "@/components/super-admin/SuperAdminLayout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Pagination, PaginationContent, PaginationItem, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
import { Label } from "@/components/ui/label";
import { getTenants, createTenant, suspendTenant, reactivateTenant, updateTenant, type Tenant, type CreateTenantPayload } from "@/services/super-admin.service";

const PAGE_SIZE = 50;

export function SuperAdminTenantsPage() {
  const navigate = useNavigate();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [editId, setEditId] = useState<number | null>(null);
  const [formData, setFormData] = useState<CreateTenantPayload>({
    name: "",
    admin_email: "",
    admin_password: "",
    admin_name: "",
  });
  const [saving, setSaving] = useState(false);
  const searchDebounce = useRef<ReturnType<typeof setTimeout>>(undefined);

  const fetchTenants = useCallback(async (off: number, searchTerm: string, shouldAbort?: () => boolean) => {
    setLoading(true);
    try {
      const res = await getTenants({ limit: PAGE_SIZE, offset: off, search: searchTerm || undefined });
      if (shouldAbort?.()) return;
      setTenants(res.data ?? []);
      setTotal(res.total);
    } catch (err) {
      if (!shouldAbort?.()) toast.error((err as Error).message);
    } finally {
      if (!shouldAbort?.()) setLoading(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      await fetchTenants(offset, search, () => cancelled);
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [offset, search, fetchTenants]);

  useEffect(() => {
    if (searchDebounce.current) clearTimeout(searchDebounce.current);
    searchDebounce.current = setTimeout(() => {
      setSearch(searchInput);
      setOffset(0);
    }, 300);
    return () => clearTimeout(searchDebounce.current);
  }, [searchInput]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await createTenant(formData);
      toast.success("Tenant created");
      setCreateOpen(false);
      setFormData({ name: "", admin_email: "", admin_password: "", admin_name: "" });
      fetchTenants(offset, search);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editId) return;
    setSaving(true);
    try {
      await updateTenant(editId, { name: formData.name });
      toast.success("Tenant updated");
      setEditId(null);
      fetchTenants(offset, search);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const openEdit = (t: Tenant) => {
    setEditId(t.id);
    setFormData({ name: t.name, admin_email: "", admin_password: "", admin_name: "" });
  };

  const handleSuspend = async (id: number) => {
    try {
      await suspendTenant(id);
      toast.success("Tenant suspended");
      fetchTenants(offset, search);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const handleReactivate = async (id: number) => {
    try {
      await reactivateTenant(id);
      toast.success("Tenant reactivated");
      fetchTenants(offset, search);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  return (
    <SuperAdminLayout title="Tenants">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>All Tenants</CardTitle>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="absolute left-2 top-2.5 size-4 text-muted-foreground" />
              <Input
                placeholder="Search tenants..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                className="pl-8 w-64"
              />
            </div>
            <Button onClick={() => { setEditId(null); setFormData({ name: "", admin_email: "", admin_password: "", admin_name: "" }); setCreateOpen(true); }}>
              <Plus className="size-4 mr-1" /> Create Tenant
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead className="text-center">Students</TableHead>
                  <TableHead className="text-center">Coaches</TableHead>
                  <TableHead className="text-center">Users</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-32 text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow><TableCell colSpan={7} className="text-center py-8">Loading...</TableCell></TableRow>
                ) : tenants.length === 0 ? (
                  <TableRow><TableCell colSpan={7} className="text-center py-8 text-muted-foreground">No tenants found</TableCell></TableRow>
                ) : tenants.map((t) => (
                  <TableRow
                    key={t.id}
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => navigate(`/super-admin/tenants/${t.id}`)}
                  >
                    <TableCell className="font-mono text-sm text-muted-foreground">{t.id}</TableCell>
                    <TableCell className="font-medium">{t.name}</TableCell>
                    <TableCell className="text-center">{t.student_count}</TableCell>
                    <TableCell className="text-center">{t.coach_count}</TableCell>
                    <TableCell className="text-center">{t.user_count}</TableCell>
                    <TableCell>
                      <Badge variant={t.suspended_at ? "destructive" : "default"}>
                        {t.suspended_at ? "Suspended" : "Active"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                      <Button variant="ghost" size="icon" className="size-8" onClick={() => openEdit(t)}>
                        <Pencil className="size-4" />
                      </Button>
                      {t.suspended_at ? (
                        <Button variant="ghost" size="icon" className="size-8" onClick={() => handleReactivate(t.id)}>
                          <RotateCcw className="size-4" />
                        </Button>
                      ) : (
                        <Button variant="ghost" size="icon" className="size-8" onClick={() => handleSuspend(t.id)}>
                          <Ban className="size-4" />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {total > PAGE_SIZE && (
            <Pagination className="mt-4">
              <PaginationContent className="flex items-center justify-between w-full">
                <p className="text-sm text-muted-foreground">
                  Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
                </p>
                <div className="flex gap-2">
                  <PaginationItem>
                    <PaginationPrevious onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))} className={offset === 0 ? "pointer-events-none opacity-50" : ""} />
                  </PaginationItem>
                  <PaginationItem>
                    <PaginationNext onClick={() => setOffset((o) => o + PAGE_SIZE)} className={offset + PAGE_SIZE >= total ? "pointer-events-none opacity-50" : ""} />
                  </PaginationItem>
                </div>
              </PaginationContent>
            </Pagination>
          )}
        </CardContent>
      </Card>

      {/* Create / Edit Tenant Dialog */}
      <Dialog open={createOpen || editId !== null} onOpenChange={(open) => { if (!open) { setCreateOpen(false); setEditId(null); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editId ? "Edit Tenant" : "Create Tenant"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={editId ? handleEdit : handleCreate} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="name">Organization Name</Label>
              <Input id="name" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required />
            </div>
            {!editId && (
              <>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="admin_name">Admin Name</Label>
                  <Input id="admin_name" value={formData.admin_name} onChange={(e) => setFormData({ ...formData, admin_name: e.target.value })} required />
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="admin_email">Admin Email</Label>
                  <Input id="admin_email" type="email" value={formData.admin_email} onChange={(e) => setFormData({ ...formData, admin_email: e.target.value })} required />
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="admin_password">Admin Password</Label>
                  <Input id="admin_password" type="password" value={formData.admin_password} onChange={(e) => setFormData({ ...formData, admin_password: e.target.value })} required />
                </div>
              </>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => { setCreateOpen(false); setEditId(null); }}>Cancel</Button>
              <Button type="submit" disabled={saving}>{saving ? "Saving..." : editId ? "Save" : "Create"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </SuperAdminLayout>
  );
}
