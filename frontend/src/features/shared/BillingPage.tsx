import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Progress } from "@/components/ui/progress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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
import {
  mockBillingPlan,
  mockUsage,
  mockBillingHistory,
} from "@/features/shared/mockData";

export function BillingPage() {
  return (
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
          <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div className="flex flex-col gap-2">
              <div className="flex items-center gap-2">
                <h3 className="text-2xl font-bold">{mockBillingPlan.name}</h3>
                <Badge>Active</Badge>
              </div>
              <p className="text-3xl font-bold">
                ${mockBillingPlan.price}
                <span className="text-sm font-normal text-muted-foreground">
                  /{mockBillingPlan.billingCycle}
                </span>
              </p>
              <p className="text-sm text-muted-foreground">
                Next billing date: {new Date(mockBillingPlan.nextBillingDate).toLocaleDateString()}
              </p>
            </div>
            <div className="flex flex-col gap-2">
              <Button className="w-fit">
                Upgrade Plan
                <ArrowUpRightIcon className="size-4 ml-1" />
              </Button>
              <Button variant="outline" className="w-fit">
                Cancel Subscription
              </Button>
            </div>
          </div>
          <Separator className="my-4" />
          <div>
            <h4 className="font-medium mb-2">Plan Features</h4>
            <ul className="grid gap-2 md:grid-cols-2">
              {mockBillingPlan.features.map((feature) => (
                <li key={feature} className="flex items-center gap-2 text-sm">
                  <CheckCircle2Icon className="size-4 text-green-500" />
                  {feature}
                </li>
              ))}
            </ul>
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
          <div className="grid gap-6 md:grid-cols-2">
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <GraduationCapIcon className="size-4 text-muted-foreground" />
                  <span className="text-sm font-medium">Students</span>
                </div>
                <span className="text-sm text-muted-foreground">
                  {mockUsage.students.used} / {mockUsage.students.limit}
                </span>
              </div>
              <Progress value={mockUsage.students.percentage} className="h-2" />
            </div>
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <UsersIcon className="size-4 text-muted-foreground" />
                  <span className="text-sm font-medium">Coaches</span>
                </div>
                <span className="text-sm text-muted-foreground">
                  {mockUsage.coaches.used} / {mockUsage.coaches.limit}
                </span>
              </div>
              <Progress value={mockUsage.coaches.percentage} className="h-2" />
            </div>
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <HardDriveIcon className="size-4 text-muted-foreground" />
                  <span className="text-sm font-medium">Storage</span>
                </div>
                <span className="text-sm text-muted-foreground">
                  {mockUsage.storage.used} {mockUsage.storage.unit} / {mockUsage.storage.limit} {mockUsage.storage.unit}
                </span>
              </div>
              <Progress value={mockUsage.storage.percentage} className="h-2" />
            </div>
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <ClipboardListIcon className="size-4 text-muted-foreground" />
                  <span className="text-sm font-medium">Tests</span>
                </div>
                <span className="text-sm text-muted-foreground">
                  {mockUsage.tests.used} / Unlimited
                </span>
              </div>
              <div className="h-2 rounded-full bg-muted" />
            </div>
          </div>
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
          <div className="rounded-lg border overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Date</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Amount</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {mockBillingHistory.map((invoice) => (
                  <TableRow key={invoice.id}>
                    <TableCell>{new Date(invoice.date).toLocaleDateString()}</TableCell>
                    <TableCell>{invoice.description}</TableCell>
                    <TableCell className="font-medium">${invoice.amount}</TableCell>
                    <TableCell>
                      <Badge variant="default" className="bg-green-500">{invoice.status}</Badge>
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
