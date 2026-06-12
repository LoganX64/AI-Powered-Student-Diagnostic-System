import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeftIcon, BarChart3Icon } from "lucide-react";
import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const DUMMY_ATTEMPTS = [
  { attempt_id: 10, test_id: 5, test_title: "Mathematics Mid-Term", sqi_score: 78.2 },
  { attempt_id: 8, test_id: 3, test_title: "Physics Chapter Test", sqi_score: 65.4 },
  { attempt_id: 5, test_id: 1, test_title: "Chemistry Quiz", sqi_score: 82.1 },
];

const AVG_SQI = DUMMY_ATTEMPTS.reduce((sum, a) => sum + a.sqi_score, 0) / DUMMY_ATTEMPTS.length;

export function StudentSQIPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const studentId = Number(id);

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <SiteHeader title="SQI Score" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

          {/* Back button */}
          <Button
            variant="ghost"
            size="sm"
            className="w-fit"
            onClick={() => navigate(`/admin/students/${studentId}`)}
          >
            <ArrowLeftIcon className="size-4 mr-2" /> Back to Student Detail
          </Button>

          {/* Header */}
          <div className="flex items-center gap-3">
            <BarChart3Icon className="size-6 text-muted-foreground" />
            <h1 className="text-2xl font-bold">SQI Score</h1>
            <Badge variant="secondary">Demo Data</Badge>
          </div>

          {/* Summary card */}
          <Card>
            <CardHeader>
              <CardTitle>Summary</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex gap-8">
                <div>
                  <p className="text-sm text-muted-foreground">Average SQI</p>
                  <p className="text-3xl font-bold">{AVG_SQI.toFixed(1)}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Total Tests</p>
                  <p className="text-3xl font-bold">{DUMMY_ATTEMPTS.length}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Per-attempt cards */}
          <div className="flex flex-col gap-3">
            <h2 className="text-base font-semibold">Attempts</h2>
            {DUMMY_ATTEMPTS.map((attempt) => (
              <Card key={attempt.attempt_id}>
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    <span>{attempt.test_title}</span>
                    <Badge variant="outline" className="font-mono">
                      SQI: {attempt.sqi_score.toFixed(1)}
                    </Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex gap-6 text-sm">
                    <div>
                      <span className="text-muted-foreground">Attempt ID: </span>
                      <span className="font-mono">{attempt.attempt_id}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Test ID: </span>
                      <span className="font-mono">{attempt.test_id}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
