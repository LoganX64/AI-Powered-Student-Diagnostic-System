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
import { useRole } from "@/hooks/useRole";

const adminCards = [
  {
    title: "Total Coaches",
    value: "12",
    change: "+2",
    trend: "up" as const,
    footer: "2 new this month",
    icon: <UsersIcon className="size-4" />,
    description: "Active coaches in system",
  },
  {
    title: "Total Students",
    value: "248",
    change: "+18",
    trend: "up" as const,
    footer: "18 enrolled this week",
    icon: <GraduationCapIcon className="size-4" />,
    description: "Across all coaches",
  },
  {
    title: "Tests Created",
    value: "89",
    change: "+12%",
    trend: "up" as const,
    footer: "Up from last month",
    icon: <ClipboardCheckIcon className="size-4" />,
    description: "Total assessments available",
  },
  {
    title: "At-Risk Students",
    value: "23",
    change: "-5",
    trend: "down" as const,
    footer: "Improvement from last week",
    icon: <AlertCircleIcon className="size-4" />,
    description: "Scored below 40% threshold",
  },
];

const coachCards = [
  {
    title: "My Students",
    value: "48",
    change: "+6",
    trend: "up" as const,
    footer: "6 new this semester",
    icon: <GraduationCapIcon className="size-4" />,
    description: "Across all active sessions",
  },
  {
    title: "Assessments Completed",
    value: "134",
    change: "+18%",
    trend: "up" as const,
    footer: "Up 18% from last month",
    icon: <ClipboardCheckIcon className="size-4" />,
    description: "Diagnostic quizzes submitted",
  },
  {
    title: "At-Risk Students",
    value: "7",
    change: "-2",
    trend: "down" as const,
    footer: "Improved from last week",
    icon: <AlertCircleIcon className="size-4" />,
    description: "Scoring below passing threshold",
  },
  {
    title: "Avg. SQI Score",
    value: "72.4",
    change: "+3.1",
    trend: "up" as const,
    footer: "Class average improving",
    icon: <TrendingUpIcon className="size-4" />,
    description: "Student Quality Index this term",
  },
];

export function DashboardSectionCards() {
  const role = useRole();
  const cards = role === "admin" ? adminCards : coachCards;

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
