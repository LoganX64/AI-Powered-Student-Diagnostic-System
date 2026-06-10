import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Trash2Icon, UserPlusIcon } from "lucide-react";
import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
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
import { createCoach, deleteCoach, getCoaches, type CreateCoachPayload, type Coach } from "@/services/admin.service";

export function CoachesPage() {
  const [coaches, setCoaches] = useState<Coach[]>([]);
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);

  useEffect(() => {
    getCoaches().then(setCoaches).catch(() => {});
  }, []);

  const handleCreate: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const data: CreateCoachPayload = {
      name: fd.get("name") as string,
      email: fd.get("email") as string,
      password: fd.get("password") as string,
    };

    try {
      setCreating(true);
      const res = await createCoach(data);
      const newCoach: Coach = {
        coach_id: res.coach_id,
        user_id: res.user_id,
        name: data.name,
        email: data.email,
      };
      setCoaches((prev) => [newCoach, ...prev]);
      toast.success(`Coach "${data.name}" created — ID: ${res.coach_id}`);
      (e.target as HTMLFormElement).reset();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (coach: Coach) => {
    try {
      setDeletingId(coach.coach_id);
      await deleteCoach(coach.coach_id);
      setCoaches((prev) => prev.filter((c) => c.coach_id !== coach.coach_id));
      toast.success(`Coach "${coach.name}" deleted`);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <SiteHeader title="Coaches" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

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
                  </div>
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
              <Badge variant="secondary">{coaches.length}</Badge>
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
                      <TableHead className="w-20 text-right">Action</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {coaches.map((coach) => (
                      <TableRow key={coach.coach_id}>
                        <TableCell className="font-mono text-sm text-muted-foreground">
                          {coach.coach_id}
                        </TableCell>
                        <TableCell className="font-medium">{coach.name}</TableCell>
                        <TableCell className="text-muted-foreground">{coach.email}</TableCell>
                        <TableCell className="text-right">
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="size-8 text-muted-foreground hover:text-destructive"
                                disabled={deletingId === coach.coach_id}
                                aria-label={`Delete ${coach.name}`}
                              >
                                <Trash2Icon className="size-4" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Delete Coach</AlertDialogTitle>
                                <AlertDialogDescription>
                                  Are you sure you want to delete{" "}
                                  <span className="font-semibold">{coach.name}</span>?
                                  This action cannot be undone.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() => handleDelete(coach)}
                                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                >
                                  Delete
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>

        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
