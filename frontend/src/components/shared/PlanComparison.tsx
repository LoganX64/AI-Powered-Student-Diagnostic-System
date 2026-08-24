import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Check } from "lucide-react";
import type { Plan } from "@/services/super-admin.service";

interface PlanComparisonProps {
  plans: Plan[];
  currentPlanId: number;
  onSelectPlan: (planSlug: string) => void;
}

function formatPrice(pricePaise: number): string {
  if (pricePaise === 0) return "Free";
  return `₹${(pricePaise / 100).toLocaleString("en-IN")}/mo`;
}

export function PlanComparison({
  plans,
  currentPlanId,
  onSelectPlan,
}: PlanComparisonProps) {
  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {plans.map((plan) => (
        <Card
          key={plan.id}
          className={plan.id === currentPlanId ? "border-primary" : ""}
        >
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>{plan.name}</CardTitle>
              {plan.id === currentPlanId && <Badge>Current</Badge>}
            </div>
            <p className="text-2xl font-bold">
              {formatPrice(plan.price_monthly)}
            </p>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 text-sm">
            <div className="flex items-center gap-2">
              <Check className="size-4 text-green-500" />
              <span>
                {plan.student_limit === -1 ? "Unlimited" : plan.student_limit}{" "}
                students
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Check className="size-4 text-green-500" />
              <span>
                {plan.coach_limit === -1 ? "Unlimited" : plan.coach_limit}{" "}
                coaches
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Check className="size-4 text-green-500" />
              <span>{plan.sqi_access ? "SQI Analytics" : "No SQI"}</span>
            </div>
            <div className="flex items-center gap-2">
              <Check className="size-4 text-green-500" />
              <span>
                {plan.video_proctoring_included
                  ? `Video Proctoring (${plan.video_proctoring_limit})`
                  : "No Video Proctoring"}
              </span>
            </div>
            {plan.id !== currentPlanId && (
              <Button className="mt-2" onClick={() => onSelectPlan(plan.slug)}>
                {plan.price_monthly > 0 ? "Upgrade" : "Downgrade"}
              </Button>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
