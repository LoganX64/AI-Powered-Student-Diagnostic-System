import { useEffect, useState, useCallback } from "react";
import { toast } from "sonner";
import { FolderIcon, PlusIcon, Trash2Icon, UsersIcon, PencilIcon } from "lucide-react";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  getBatches,
  createBatch,
  deleteBatch,
  updateBatch,
  transferStudentBatch,
  getStudents,
  type Batch,
  type Student,
} from "@/services/dashboard.service";

export function BatchesPage() {
  const PAGE_SIZE = 50;

  const [batches, setBatches] = useState<Batch[]>([]);
  const [members, setMembers] = useState<Student[]>([]);
  const [memberTotal, setMemberTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  const [newName, setNewName] = useState("");
  const [creating, setCreating] = useState(false);

  const [selectedBatchId, setSelectedBatchId] = useState("");
  const [memberOffset, setMemberOffset] = useState(0);
  const [deleteTarget, setDeleteTarget] = useState<Batch | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [editTarget, setEditTarget] = useState<Batch | null>(null);
  const [editName, setEditName] = useState("");
  const [editing, setEditing] = useState(false);
  const [transferringId, setTransferringId] = useState<number | null>(null);
  const [pendingTransfer, setPendingTransfer] = useState<{
    studentId: number;
    studentName: string;
    targetBatchId: number | null;
    targetBatchName: string;
  } | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const batchRes = await getBatches();
      setBatches(batchRes.data ?? []);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchMembers = useCallback(async (batchId: string, off: number) => {
    if (!batchId) return;
    try {
      const res = await getStudents({
        limit: PAGE_SIZE,
        offset: off,
        batch_id: Number(batchId),
      });
      setMembers(res.data ?? []);
      setMemberTotal(res.total);
    } catch (err) {
      toast.error((err as Error).message);
    }
  }, []);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const batchRes = await getBatches();
        if (!active) return;
        setBatches(batchRes.data ?? []);
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

  useEffect(() => {
    if (!selectedBatchId) return;
    getStudents({ limit: PAGE_SIZE, offset: memberOffset, batch_id: Number(selectedBatchId) })
      .then((res) => {
        setMembers(res.data ?? []);
        setMemberTotal(res.total);
      })
      .catch((err) => toast.error((err as Error).message));
  }, [selectedBatchId, memberOffset]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    const name = newName.trim();
    if (!name) return;
    try {
      setCreating(true);
      await createBatch(name);
      toast.success(`Batch "${name}" created`);
      setNewName("");
      await refresh();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      setDeleting(true);
      const res = await deleteBatch(deleteTarget.id);
      toast.success(
        `Batch deleted. ${res.students_reassigned} student(s) unassigned.`
      );
      if (String(deleteTarget.id) === selectedBatchId) setSelectedBatchId("");
      await refresh();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setDeleting(false);
      setDeleteTarget(null);
    }
  };

  const openEdit = (batch: Batch) => {
    setEditTarget(batch);
    setEditName(batch.name);
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editTarget) return;
    const name = editName.trim();
    if (!name) return;
    try {
      setEditing(true);
      await updateBatch(editTarget.id, name);
      toast.success(`Batch renamed to "${name}"`);
      setEditTarget(null);
      await refresh();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setEditing(false);
    }
  };

  const handleTransfer = (studentId: number, value: string) => {
    const target = value === "none" ? null : Number(value);
    const student = members.find((m) => m.student_id === studentId);
    const targetBatch = target !== null ? batches.find((b) => b.id === target) : null;
    setPendingTransfer({
      studentId,
      studentName: student?.name ?? "",
      targetBatchId: target,
      targetBatchName: targetBatch?.name ?? "No batch",
    });
  };

  const confirmTransfer = async () => {
    if (!pendingTransfer) return;
    const { studentId, targetBatchId } = pendingTransfer;
    try {
      setTransferringId(studentId);
      await transferStudentBatch(studentId, targetBatchId);
      toast.success("Student batch updated");
      await refresh();
      fetchMembers(selectedBatchId, memberOffset);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setTransferringId(null);
      setPendingTransfer(null);
    }
  };

  return (
    <DashboardLayout title="Batches">
      {/* Create batch */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <PlusIcon className="size-5" />
            Create Batch
          </CardTitle>
          <CardDescription>
            Batches are organization-wide. Any admin or coach in your organization can manage them.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreate} className="flex flex-col gap-4 sm:flex-row sm:items-end">
            <div className="flex flex-1 flex-col gap-2">
              <Label htmlFor="batch-name">Batch name</Label>
              <Input
                id="batch-name"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="e.g. Batch 2025-A"
                required
              />
            </div>
            <Button type="submit" disabled={creating} className="w-fit">
              {creating ? "Creating…" : "Create Batch"}
            </Button>
          </form>
        </CardContent>
      </Card>

      {/* Batch list */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FolderIcon className="size-5" />
            All Batches
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : batches.length === 0 ? (
            <p className="text-sm text-muted-foreground">No batches yet. Create one above.</p>
          ) : (
            <div className="rounded-lg border overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead className="w-32">Students</TableHead>
                    <TableHead className="w-44">Created</TableHead>
                    <TableHead className="w-24 text-right">Action</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {batches.map((b) => (
                    <TableRow key={b.id}>
                      <TableCell className="font-medium">{b.name}</TableCell>
                      <TableCell>
                        <Badge variant="secondary">{b.student_count}</Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {new Date(b.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="size-8 text-muted-foreground hover:text-foreground"
                            aria-label={`Edit batch ${b.name}`}
                            onClick={() => openEdit(b)}
                          >
                            <PencilIcon className="size-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="size-8 text-muted-foreground hover:text-destructive"
                            disabled={deleting}
                            aria-label={`Delete batch ${b.name}`}
                            onClick={() => setDeleteTarget(b)}
                          >
                            <Trash2Icon className="size-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Member management */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <UsersIcon className="size-5" />
            Manage Members
          </CardTitle>
          <CardDescription>
            Select a batch to view and move its students between batches.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-2 sm:max-w-xs">
            <Label>Batch</Label>
            <Select
              value={selectedBatchId}
              onValueChange={(v) => {
                setSelectedBatchId(v);
                setMemberOffset(0);
              }}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select a batch" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {batches.map((b) => (
                    <SelectItem key={b.id} value={b.id.toString()}>
                      {b.name}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          {selectedBatchId && (
            members.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No students in this batch yet.
              </p>
            ) : (
              <>
                <div className="rounded-lg border overflow-hidden">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-16">ID</TableHead>
                        <TableHead>Name</TableHead>
                        <TableHead>Code</TableHead>
                        <TableHead className="w-64">Move to</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {members.map((s) => (
                        <TableRow key={s.student_id}>
                          <TableCell className="font-mono text-sm text-muted-foreground">
                            {s.student_id}
                          </TableCell>
                          <TableCell className="font-medium">{s.name}</TableCell>
                          <TableCell>
                            <Badge variant="outline" className="font-mono">
                              {s.student_code}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <Select
                              value={selectedBatchId}
                              disabled={transferringId === s.student_id}
                              onValueChange={(v) => handleTransfer(s.student_id, v)}
                            >
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectGroup>
                                  <SelectItem value="none">No batch</SelectItem>
                                  {batches
                                    .filter((b) => String(b.id) !== selectedBatchId)
                                    .map((b) => (
                                      <SelectItem key={b.id} value={b.id.toString()}>
                                        {b.name}
                                      </SelectItem>
                                    ))}
                                </SelectGroup>
                              </SelectContent>
                            </Select>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
                {memberTotal > PAGE_SIZE && (
                  <Pagination>
                    <PaginationContent className="flex items-center justify-between w-full">
                      <p className="text-sm text-muted-foreground">
                        Showing {memberOffset + 1}–{Math.min(memberOffset + PAGE_SIZE, memberTotal)} of {memberTotal}
                      </p>
                      <div className="flex gap-2">
                        <PaginationItem>
                          <PaginationPrevious
                            onClick={() => setMemberOffset((o) => Math.max(0, o - PAGE_SIZE))}
                            className={memberOffset === 0 ? "pointer-events-none opacity-50" : ""}
                          />
                        </PaginationItem>
                        <PaginationItem>
                          <PaginationNext
                            onClick={() => setMemberOffset((o) => o + PAGE_SIZE)}
                            className={memberOffset + PAGE_SIZE >= memberTotal ? "pointer-events-none opacity-50" : ""}
                          />
                        </PaginationItem>
                      </div>
                    </PaginationContent>
                  </Pagination>
                )}
              </>
            )
          )}
        </CardContent>
      </Card>

      <AlertDialog open={deleteTarget !== null} onOpenChange={(o) => !o && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Batch</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the batch{" "}
              <span className="font-semibold">{deleteTarget?.name}</span>? Its students will be
              unassigned (not deleted).
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete Batch
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={editTarget !== null} onOpenChange={(o) => !o && setEditTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Batch</DialogTitle>
            <DialogDescription>
              Rename the batch <span className="font-semibold">{editTarget?.name}</span>.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleEdit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="edit-batch-name">Batch name</Label>
              <Input
                id="edit-batch-name"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                placeholder="e.g. Batch 2025-A"
                required
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditTarget(null)}>
                Cancel
              </Button>
              <Button type="submit" disabled={editing}>
                {editing ? "Saving…" : "Save"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={pendingTransfer !== null} onOpenChange={(o) => !o && setPendingTransfer(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Move Student</AlertDialogTitle>
            <AlertDialogDescription>
              Move <span className="font-semibold">{pendingTransfer?.studentName}</span> to{" "}
              <span className="font-semibold">{pendingTransfer?.targetBatchName}</span>?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmTransfer}>
              Move Student
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </DashboardLayout>
  );
}
