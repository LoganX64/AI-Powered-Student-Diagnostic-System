import { useEffect, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Trash2Icon, UserPlusIcon, RotateCcwIcon } from "lucide-react";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Pagination, PaginationContent, PaginationItem, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
import { createCoach, deleteCoach, reactivateCoach, getCoaches, type Coach } from "@/services/dashboard.service";
import { createCoachSchema, zodErrors } from "@/lib/validations";
import { SubjectPicker } from "@/components/admin/forms/SubjectPicker";

const PAGE_SIZE = 50;

export function CoachesPage() {
  const navigate = useNavigate();
  const [coaches, setCoaches] = useState<Coach[]>([]);
  const [creating, setCreating] = useState(false);
  const [createErrors, setCreateErrors] = useState<Record<string, string>>({});
  const [deactivatingId, setDeactivatingId] = useState<number | null>(null);
  const [reactivatingId, setReactivatingId] = useState<number | null>(null);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [includeDeactivated, setIncludeDeactivated] = useState(false);
  const [selectedSubjectIds, setSelectedSubjectIds] = useState<number[]>([]);

  const fetchCoaches = useCallback(async (off: number, deactivated: boolean) => {
    try {
      const res = await getCoaches({ limit: PAGE_SIZE, offset: off, include_deactivated: deactivated });
      setCoaches(res.data ?? []);
      setTotal(res.total);
    } catch {
      // silently ignore
    }
  }, []);

  useEffect(() => {
    fetchCoaches(offset, includeDeactivated);
  }, [offset, includeDeactivated, fetchCoaches]);

  const handleCreate: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const raw = {
      name: fd.get("name") as string,
      email: fd.get("email") as string,
      password: fd.get("password") as string,
      subject_ids: selectedSubjectIds,
    };

    const result = createCoachSchema.safeParse(raw);
    if (!result.success) {
      setCreateErrors(zodErrors(result.error));
      return;
    }
    setCreateErrors({});

    try {
      setCreating(true);
      const res = await createCoach(result.data);
      toast.success(`Coach "${result.data.name}" created — ID: ${res.coach_id}`);
      (e.target as HTMLFormElement).reset();
      setSelectedSubjectIds([]);
      fetchCoaches(offset, includeDeactivated);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setCreating(false);
    }
  };

  const handleDeactivate = async (coach: Coach) => {
    try {
      setDeactivatingId(coach.coach_id);
      await deleteCoach(coach.coach_id);
      toast.success(`Coach "${coach.name}" account deactivated`);
      fetchCoaches(offset, includeDeactivated);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setDeactivatingId(null);
    }
  };

  const handleReactivate = async (coach: Coach) => {
    try {
      setReactivatingId(coach.coach_id);
      await reactivateCoach(coach.coach_id);
      toast.success(`Coach "${coach.name}" account reactivated`);
      fetchCoaches(offset, includeDeactivated);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setReactivatingId(null);
    }
  };

  return (
    <DashboardLayout title="Coaches">
      {/* Create form */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <UserPlusIcon className="size-5" />
            Create Coach
          </CardTitle>
          <CardDescription>
            Add a new coach to your organization.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreate} className="flex flex-col gap-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="flex flex-col gap-2">
                <Label htmlFor="coach-name">Full Name</Label>
                <Input
                  id="coach-name"
                  name="name"
                  placeholder="John Smith"
                  required
                />
                {createErrors.name && <p className="text-sm text-destructive">{createErrors.name}</p>}
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="coach-email">Email</Label>
                <Input
                  id="coach-email"
                  name="email"
                  type="email"
                  placeholder="coach@academy.com"
                  required
                />
                {createErrors.email && <p className="text-sm text-destructive">{createErrors.email}</p>}
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="coach-password">Password</Label>
                <Input
                  id="coach-password"
                  name="password"
                  type="password"
                  placeholder="Min. 8 characters"
                  required
                  minLength={8}
                />
                {createErrors.password && <p className="text-sm text-destructive">{createErrors.password}</p>}
              </div>
            </div>
            <div className="flex flex-col gap-2">
              <Label>Subjects *</Label>
              <SubjectPicker
                selected={selectedSubjectIds}
                onChange={setSelectedSubjectIds}
                error={createErrors.subject_ids}
              />
            </div>
            <Button type="submit" disabled={creating} className="w-fit">
              {creating ? "Creating…" : "Create Coach"}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Separator />

      {/* Coaches table */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">
            All Coaches
          </h2>
          <div className="flex items-center gap-3">
            <Button
              variant={includeDeactivated ? "default" : "outline"}
              size="sm"
              onClick={() => {
                setIncludeDeactivated(!includeDeactivated);
                setOffset(0);
              }}
            >
              {includeDeactivated ? "Showing Deactivated" : "Show Deactivated"}
            </Button>
            <Badge variant="secondary">{total}</Badge>
          </div>
        </div>

        {coaches.length === 0 ? (
          <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">
              No coaches yet. Create one above.
            </p>
          </div>
        ) : (
          <div className="rounded-lg border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Subjects</TableHead>
                  <TableHead className="w-20 text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {coaches.map((coach) => (
                  <TableRow
                    key={coach.coach_id}
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => navigate(`/admin/coaches/${coach.coach_id}`)}
                  >
                    <TableCell className="font-mono text-sm text-muted-foreground">
                      {coach.coach_id}
                    </TableCell>
                    <TableCell className="font-medium">{coach.name}</TableCell>
                    <TableCell className="text-muted-foreground">{coach.email}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {(coach.subjects ?? []).length === 0 ? (
                          <span className="text-xs text-muted-foreground">—</span>
                        ) : (
                          (coach.subjects ?? []).map((s) => (
                            <Badge key={s.subject_id} variant="secondary" className="text-xs">
                              {s.name}
                            </Badge>
                          ))
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      {coach.deleted_at ? (
                         <span onClick={(e) => e.stopPropagation()}>
                           <AlertDialog>
                             <AlertDialogTrigger asChild>
                               <Button
                                 variant="ghost"
                                 size="icon"
                                 className="size-8 text-muted-foreground hover:text-green-600"
                                 disabled={reactivatingId === coach.coach_id}
                                 aria-label={`Reactivate account for ${coach.name}`}
                               >
                                 <RotateCcwIcon className="size-4" />
                               </Button>
                             </AlertDialogTrigger>
                             <AlertDialogContent>
                               <AlertDialogHeader>
                                 <AlertDialogTitle>Reactivate Account</AlertDialogTitle>
                                 <AlertDialogDescription>
                                   Are you sure you want to reactivate the account for{" "}
                                   <span className="font-semibold">{coach.name}</span>?
                                 </AlertDialogDescription>
                               </AlertDialogHeader>
                               <AlertDialogFooter>
                                 <AlertDialogCancel>Cancel</AlertDialogCancel>
                                 <AlertDialogAction
                                   onClick={() => handleReactivate(coach)}
                                   className="bg-green-600 text-white hover:bg-green-700"
                                 >
                                   Reactivate Account
                                 </AlertDialogAction>
                               </AlertDialogFooter>
                             </AlertDialogContent>
                           </AlertDialog>
                         </span>
                      ) : (
                         <span onClick={(e) => e.stopPropagation()}>
                           <AlertDialog>
                             <AlertDialogTrigger asChild>
                               <Button
                                 variant="ghost"
                                 size="icon"
                                 className="size-8 text-muted-foreground hover:text-destructive"
                                 disabled={deactivatingId === coach.coach_id}
                                 aria-label={`Deactivate account for ${coach.name}`}
                               >
                                 <Trash2Icon className="size-4" />
                               </Button>
                             </AlertDialogTrigger>
                             <AlertDialogContent>
                               <AlertDialogHeader>
                                 <AlertDialogTitle>Deactivate Account</AlertDialogTitle>
                                 <AlertDialogDescription>
                                   Are you sure you want to deactivate the account for{" "}
                                   <span className="font-semibold">{coach.name}</span>?
                                 </AlertDialogDescription>
                               </AlertDialogHeader>
                               <AlertDialogFooter>
                                 <AlertDialogCancel>Cancel</AlertDialogCancel>
                                 <AlertDialogAction
                                   onClick={() => handleDeactivate(coach)}
                                   className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                 >
                                   Deactivate Account
                                 </AlertDialogAction>
                               </AlertDialogFooter>
                             </AlertDialogContent>
                           </AlertDialog>
                         </span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        {/* Pagination */}
        {total > PAGE_SIZE && (
          <Pagination>
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
      </div>
    </DashboardLayout>
  );
}
