import {
  BarChart3Icon,
  BrainCircuitIcon,
  ClockIcon,
  TargetIcon,
  TrendingUpIcon,
  TrendingDownIcon,
  AlertTriangleIcon,
  CheckCircle2Icon,
  ZapIcon,
  EyeOffIcon,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
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
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { SQIAnalysis } from "@/services/types";

function scoreColor(score: number): string {
  if (score >= 75) return "text-green-600";
  if (score >= 50) return "text-yellow-600";
  return "text-red-600";
}

function progressColor(score: number): string {
  if (score >= 70) return "bg-green-500";
  if (score >= 50) return "bg-yellow-500";
  return "bg-red-500";
}

function statusBadge(status: string) {
  const map: Record<string, { label: string; className: string }> = {
    mastered: { label: "Mastered", className: "bg-green-100 text-green-700 border-green-300 dark:bg-green-900/30 dark:text-green-400" },
    almost_there: { label: "Almost There", className: "bg-yellow-100 text-yellow-700 border-yellow-300 dark:bg-yellow-900/30 dark:text-yellow-400" },
    confused: { label: "Confused", className: "bg-red-100 text-red-700 border-red-300 dark:bg-red-900/30 dark:text-red-400" },
    not_studied: { label: "Not Studied", className: "bg-gray-100 text-gray-600 border-gray-300 dark:bg-gray-800 dark:text-gray-400" },
    not_reached: { label: "Not Reached", className: "bg-gray-100 text-gray-600 border-gray-300 dark:bg-gray-800 dark:text-gray-400" },
  };
  const { label, className } = map[status] ?? { label: status, className: "" };
  return <Badge variant="outline" className={className}>{label}</Badge>;
}

function formatFlagName(key: string): string {
  return key
    .replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

// ─── Sub-components ──────────────────────────────────────────────────────────

function SummaryRow({ data }: { data: SQIAnalysis }) {
  const s = data.exam_summary;
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <Card>
        <CardContent className="p-4 text-center">
          <p className="text-xs text-muted-foreground">Overall SQI</p>
          <p className={`text-3xl font-bold tabular-nums ${scoreColor(data.overall_sqi)}`}>
            {data.overall_sqi.toFixed(1)}
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4 text-center">
          <p className="text-xs text-muted-foreground">Accuracy</p>
          <p className="text-3xl font-bold tabular-nums">
            {s.correct}/{s.attempted}
          </p>
          <p className={`text-xs ${scoreColor(s.score_percent)}`}>
            {s.score_percent.toFixed(1)}%
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4 text-center">
          <p className="text-xs text-muted-foreground">Net Score</p>
          <p className="text-3xl font-bold tabular-nums">
            {s.net_score.toFixed(1)}
          </p>
          <p className="text-xs text-muted-foreground">
            of {s.max_possible_score.toFixed(1)} max
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4 text-center">
          <p className="text-xs text-muted-foreground">Skipped / Unseen</p>
          <p className="text-3xl font-bold tabular-nums text-yellow-600">
            {s.skipped + s.unseen}
          </p>
          <p className="text-xs text-muted-foreground">
            {s.skipped} skipped, {s.unseen} unseen
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function DimensionBars({ data }: { data: SQIAnalysis }) {
  const dims = [
    { label: "Mastery", score: data.dimensions.mastery, icon: <BrainCircuitIcon className="size-4" /> },
    { label: "Speed", score: data.dimensions.speed, icon: <ClockIcon className="size-4" /> },
    { label: "Risk", score: data.dimensions.risk, icon: <TargetIcon className="size-4" /> },
    { label: "Coverage", score: data.dimensions.coverage, icon: <BarChart3Icon className="size-4" /> },
  ];

  return (
    <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
      {dims.map((d) => (
        <Card key={d.label}>
          <CardContent className="p-4">
            <div className="mb-2 flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm font-medium">
                {d.icon} {d.label}
              </div>
              <span className={`text-lg font-bold tabular-nums ${scoreColor(d.score)}`}>
                {d.score.toFixed(1)}
              </span>
            </div>
            <Progress value={d.score} className="h-2" />
            <div
              className={`mt-1 h-2 rounded-full ${progressColor(d.score)}`}
              style={{ width: `${d.score}%` }}
            />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function ConceptTable({ data }: { data: SQIAnalysis }) {
  if (!data.concept_profiles || data.concept_profiles.length === 0) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-muted-foreground">
          No concept data available.
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Concept Breakdown</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="rounded-lg border overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Concept</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-center">Accuracy</TableHead>
                <TableHead className="text-center">Questions</TableHead>
                <TableHead className="text-center">Priority</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.concept_profiles.map((cp) => (
                <TableRow key={cp.concept_tag}>
                  <TableCell>
                    <div>
                      <span className="font-medium">{cp.concept_tag}</span>
                      {cp.subject && (
                        <span className="ml-2 text-xs text-muted-foreground">{cp.subject}</span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>{statusBadge(cp.status)}</TableCell>
                  <TableCell className="text-center">
                    <span className={`tabular-nums ${scoreColor(cp.evidence.accuracy_pct)}`}>
                      {cp.evidence.accuracy_pct.toFixed(0)}%
                    </span>
                    <span className="text-xs text-muted-foreground ml-1">
                      ({cp.evidence.correct}/{cp.evidence.attempted})
                    </span>
                  </TableCell>
                  <TableCell className="text-center tabular-nums text-sm">
                    {cp.evidence.total_questions}
                  </TableCell>
                  <TableCell className="text-center">
                    <Badge variant="outline" className="tabular-nums">
                      #{cp.priority_rank}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}

function BehaviorFlags({ data }: { data: SQIAnalysis }) {
  const detected = Object.entries(data.behavior_flags).filter(
    ([, flag]) => flag.detected,
  );

  if (detected.length === 0) {
    return (
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CheckCircle2Icon className="size-4 text-green-600" />
            No concerning behaviors detected.
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Behavioral Flags</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex flex-wrap gap-2">
          {detected.map(([key, flag]) => (
            <Tooltip key={key}>
              <TooltipTrigger>
                <Badge variant="outline" className="gap-1 bg-orange-50 text-orange-700 border-orange-300 dark:bg-orange-900/20 dark:text-orange-400">
                  <AlertTriangleIcon className="size-3" />
                  {formatFlagName(key)}
                  <span className="text-xs opacity-70">
                    ({(flag.confidence * 100).toFixed(0)}%)
                  </span>
                </Badge>
              </TooltipTrigger>
              <TooltipContent>
                <p className="max-w-xs text-sm">{flag.evidence}</p>
              </TooltipContent>
            </Tooltip>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function HalfSplit({ data }: { data: SQIAnalysis }) {
  return (
    <div className="grid grid-cols-2 gap-3">
      <Card>
        <CardContent className="p-4 text-center">
          <div className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
            <TrendingUpIcon className="size-3.5" /> First Half
          </div>
          <p className={`mt-1 text-2xl font-bold tabular-nums ${scoreColor(data.first_half_accuracy)}`}>
            {data.first_half_accuracy.toFixed(1)}%
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4 text-center">
          <div className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
            <TrendingDownIcon className="size-3.5" /> Second Half
          </div>
          <p className={`mt-1 text-2xl font-bold tabular-nums ${scoreColor(data.second_half_accuracy)}`}>
            {data.second_half_accuracy.toFixed(1)}%
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function AttemptProfile({ data }: { data: SQIAnalysis }) {
  const p = data.attempt_profile;
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Attempt Profile</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p className="text-muted-foreground mb-1">Correct</p>
            <div className="space-y-1">
              <div className="flex justify-between">
                <span className="flex items-center gap-1"><ZapIcon className="size-3 text-green-600" /> Guessed Right</span>
                <span className="tabular-nums font-medium">{p.guessed_right}</span>
              </div>
              <div className="flex justify-between">
                <span className="flex items-center gap-1"><CheckCircle2Icon className="size-3 text-green-600" /> Carefully Right</span>
                <span className="tabular-nums font-medium">{p.carefully_right}</span>
              </div>
            </div>
          </div>
          <div>
            <p className="text-muted-foreground mb-1">Wrong</p>
            <div className="space-y-1">
              <div className="flex justify-between">
                <span className="flex items-center gap-1"><ZapIcon className="size-3 text-red-600" /> Guessed Wrong</span>
                <span className="tabular-nums font-medium">{p.guessed_wrong}</span>
              </div>
              <div className="flex justify-between">
                <span className="flex items-center gap-1"><AlertTriangleIcon className="size-3 text-red-600" /> Carefully Wrong</span>
                <span className="tabular-nums font-medium">{p.carefully_wrong}</span>
              </div>
            </div>
          </div>
          <div>
            <p className="text-muted-foreground mb-1">Skipped</p>
            <div className="flex justify-between">
              <span className="flex items-center gap-1"><EyeOffIcon className="size-3 text-muted-foreground" /> Seen & Abandoned</span>
              <span className="tabular-nums font-medium">{p.seen_abandoned}</span>
            </div>
          </div>
          <div>
            <p className="text-muted-foreground mb-1">Unseen</p>
            <div className="flex justify-between">
              <span className="flex items-center gap-1"><EyeOffIcon className="size-3 text-muted-foreground" /> Never Reached</span>
              <span className="tabular-nums font-medium">{p.never_reached}</span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// ─── Main Component ──────────────────────────────────────────────────────────

export function AnalysisDashboard({ data }: { data: SQIAnalysis }) {
  return (
    <div className="flex flex-col gap-4 rounded-lg border border-border bg-card p-4">
      <SummaryRow data={data} />
      <DimensionBars data={data} />
      <ConceptTable data={data} />
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <BehaviorFlags data={data} />
        <HalfSplit data={data} />
      </div>
      <AttemptProfile data={data} />
    </div>
  );
}
