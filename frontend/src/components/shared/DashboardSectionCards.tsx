import { TrendingUpIcon, TrendingDownIcon, UsersIcon, GraduationCapIcon, AlertCircleIcon, ClipboardCheckIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardHeader,
  CardDescription,
  CardTitle,
  CardAction,
  CardFooter,
} from "@/components/ui/card";
import { useDashboard } from "@/contexts/DashboardContext";

function buildAdminCards(counts: { totalCoaches: number; totalStudents: number; testsCreated: number }, atRiskCount: number) {
  return [
    {
      title: "Total Coaches",
      value: String(counts.totalCoaches),
      change: "",
      trend: "up" as const,
      footer: "",
      icon: <UsersIcon className="size-4" />,
      description: "Active coaches in system",
    },
    {
      title: "Total Students",
      value: String(counts.totalStudents),
      change: "",
      trend: "up" as const,
      footer: "",
      icon: <GraduationCapIcon className="size-4" />,
      description: "Across all coaches",
    },
    {
      title: "Tests Created",
      value: String(counts.testsCreated),
      change: "",
      trend: "up" as const,
      footer: "",
      icon: <ClipboardCheckIcon className="size-4" />,
      description: "Total assessments available",
    },
    {
      title: "At-Risk Students",
      value: atRiskCount > 0 ? String(atRiskCount) : "--",
      change: "",
      trend: "down" as const,
      footer: "",
      icon: <AlertCircleIcon className="size-4" />,
      description: "Scored below 40% threshold",
    },
  ];
}

function buildCoachCards(counts: { totalCoaches: number; totalStudents: number; testsCreated: number }, atRiskCount: number, avgSqi: number) {
  return [
    {
      title: "My Students",
      value: String(counts.totalStudents),
      change: "",
      trend: "up" as const,
      footer: "",
      icon: <GraduationCapIcon className="size-4" />,
      description: "Across all active sessions",
    },
    {
      title: "Assessments Completed",
      value: "--",
      change: "",
      trend: "up" as const,
      footer: "",
      icon: <ClipboardCheckIcon className="size-4" />,
      description: "Diagnostic quizzes submitted",
    },
    {
      title: "At-Risk Students",
      value: atRiskCount > 0 ? String(atRiskCount) : "--",
      change: "",
      trend: "down" as const,
      footer: "",
      icon: <AlertCircleIcon className="size-4" />,
      description: "Scoring below passing threshold",
    },
    {
      title: "Avg. SQI Score",
      value: avgSqi > 0 ? avgSqi.toFixed(1) : "--",
      change: "",
      trend: "up" as const,
      footer: "",
      icon: <TrendingUpIcon className="size-4" />,
      description: "Student Quality Index this term",
    },
  ];
}

export function DashboardSectionCards() {
  const { counts, studentsWithSQI, role } = useDashboard();

  let atRiskCount = 0;
  let totalSqi = 0;
  let sqiCount = 0;

  for (const s of studentsWithSQI) {
    if (s.average_sqi > 0 && s.average_sqi < 55) atRiskCount++;
    if (s.average_sqi > 0) {
      totalSqi += s.average_sqi;
      sqiCount++;
    }
  }

  const avgSqi = sqiCount > 0 ? totalSqi / sqiCount : 0;
  const cards = role === "admin" ? buildAdminCards(counts, atRiskCount) : buildCoachCards(counts, atRiskCount, avgSqi);

  return (
    <div className="grid grid-cols-1 gap-4 px-4 *:data-[slot=card]:bg-linear-to-t *:data-[slot=card]:from-primary/5 *:data-[slot=card]:to-card *:data-[slot=card]:shadow-xs lg:px-6 @xl/main:grid-cols-2 @5xl/main:grid-cols-4 dark:*:data-[slot=card]:bg-card">
      {cards.map((card) => (
        <Card key={card.title} className="@container/card">
          <CardHeader>
            <CardDescription>{card.title}</CardDescription>
            <CardTitle className="text-2xl font-semibold tabular-nums @[250px]/card:text-3xl">
              {card.value}
            </CardTitle>
            <CardAction>
              <Badge variant="outline">
                {card.trend === "up" ? <TrendingUpIcon /> : <TrendingDownIcon />}
                {card.change}
              </Badge>
            </CardAction>
          </CardHeader>
          <CardFooter className="flex-col items-start gap-1.5 text-sm">
            <div className="line-clamp-1 flex gap-2 font-medium">
              {card.footer} {card.icon}
            </div>
            <div className="text-muted-foreground">{card.description}</div>
          </CardFooter>
        </Card>
      ))}
    </div>
  );
}
