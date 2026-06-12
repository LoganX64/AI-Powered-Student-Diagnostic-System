import { useState } from "react";
import { toast } from "sonner";
import { useRole } from "@/hooks/useRole";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
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
import {
  CircleUserRoundIcon,
  KeyIcon,
  MonitorIcon,
  ShieldCheckIcon,
  CopyIcon,
  Trash2Icon,
} from "lucide-react";
import {
  mockAccountInfo,
  mockApiKeys,
  mockSessions,
} from "@/features/shared/mockData";

export function AccountsPage() {
  const role = useRole();
  const [apiKeys, setApiKeys] = useState(mockApiKeys);
  const [sessions, setSessions] = useState(mockSessions);

  const handleCopyKey = (prefix: string) => {
    navigator.clipboard.writeText(prefix);
    toast.success("API key copied to clipboard");
  };

  const handleRevokeKey = (keyId: string) => {
    setApiKeys(apiKeys.filter((k) => k.id !== keyId));
    toast.success("API key revoked");
  };

  const handleRevokeSession = (sessionId: string) => {
    setSessions(sessions.filter((s) => s.id !== sessionId));
    toast.success("Session revoked");
  };

  return (
    <div className="flex flex-col gap-6">
      {/* Account Info */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CircleUserRoundIcon className="size-5" />
            Account Information
          </CardTitle>
          <CardDescription>
            Your account details and status.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2 max-w-2xl">
            <div className="flex flex-col gap-1">
              <Label className="text-muted-foreground">Name</Label>
              <p className="font-medium">{mockAccountInfo.name}</p>
            </div>
            <div className="flex flex-col gap-1">
              <Label className="text-muted-foreground">Email</Label>
              <p className="font-medium">{mockAccountInfo.email}</p>
            </div>
            <div className="flex flex-col gap-1">
              <Label className="text-muted-foreground">Role</Label>
              <Badge variant="secondary" className="w-fit">{mockAccountInfo.role}</Badge>
            </div>
            <div className="flex flex-col gap-1">
              <Label className="text-muted-foreground">Status</Label>
              <Badge variant="default" className="w-fit bg-green-500">{mockAccountInfo.status}</Badge>
            </div>
            <div className="flex flex-col gap-1">
              <Label className="text-muted-foreground">Joined</Label>
              <p className="font-medium">
                {new Date(mockAccountInfo.joinedAt).toLocaleDateString()}
              </p>
            </div>
            <div className="flex flex-col gap-1">
              <Label className="text-muted-foreground">Last Login</Label>
              <p className="font-medium">
                {new Date(mockAccountInfo.lastLogin).toLocaleString()}
              </p>
            </div>
            <div className="flex flex-col gap-1">
              <Label className="text-muted-foreground">Email Verified</Label>
              <Badge variant={mockAccountInfo.emailVerified ? "default" : "destructive"} className="w-fit">
                {mockAccountInfo.emailVerified ? "Verified" : "Not Verified"}
              </Badge>
            </div>
            <div className="flex flex-col gap-1">
              <Label className="text-muted-foreground">Two-Factor Auth</Label>
              <Badge variant={mockAccountInfo.twoFactorEnabled ? "default" : "outline"} className="w-fit">
                {mockAccountInfo.twoFactorEnabled ? "Enabled" : "Disabled"}
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* API Keys */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <KeyIcon className="size-5" />
            API Keys
          </CardTitle>
          <CardDescription>
            Manage your API keys for programmatic access.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Key</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Last Used</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.map((key) => (
                  <TableRow key={key.id}>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell className="font-mono text-sm">{key.prefix}</TableCell>
                    <TableCell>{new Date(key.createdAt).toLocaleDateString()}</TableCell>
                    <TableCell>{new Date(key.lastUsed).toLocaleDateString()}</TableCell>
                    <TableCell>
                      <Badge variant="default" className="bg-green-500">{key.status}</Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex gap-1 justify-end">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-8"
                          onClick={() => handleCopyKey(key.prefix)}
                        >
                          <CopyIcon className="size-4" />
                        </Button>
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="size-8 text-destructive"
                            >
                              <Trash2Icon className="size-4" />
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Revoke API Key</AlertDialogTitle>
                              <AlertDialogDescription>
                                Are you sure you want to revoke "{key.name}"? This action cannot be undone.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => handleRevokeKey(key.id)}
                                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                              >
                                Revoke
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Active Sessions */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MonitorIcon className="size-5" />
            Active Sessions
          </CardTitle>
          <CardDescription>
            Manage your active login sessions.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Device</TableHead>
                  <TableHead>IP Address</TableHead>
                  <TableHead>Last Active</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sessions.map((session) => (
                  <TableRow key={session.id}>
                    <TableCell className="font-medium">
                      <div className="flex items-center gap-2">
                        {session.device}
                        {session.current && (
                          <Badge variant="secondary" className="text-xs">Current</Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-sm">{session.ip}</TableCell>
                    <TableCell>{new Date(session.lastActive).toLocaleString()}</TableCell>
                    <TableCell className="text-right">
                      {!session.current && (
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="size-8 text-destructive"
                            >
                              <Trash2Icon className="size-4" />
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Revoke Session</AlertDialogTitle>
                              <AlertDialogDescription>
                                Are you sure you want to revoke this session? The user will be logged out.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => handleRevokeSession(session.id)}
                                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                              >
                                Revoke
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
