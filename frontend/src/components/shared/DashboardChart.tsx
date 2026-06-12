"use client";

import * as React from "react";
import { Area, AreaChart, CartesianGrid, XAxis } from "recharts";
import { useIsMobile } from "@/hooks/use-mobile";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

const chartData = [
  { date: "2024-01-15", avgScore: 61, passing: 30 },
  { date: "2024-01-22", avgScore: 63, passing: 31 },
  { date: "2024-01-29", avgScore: 60, passing: 29 },
  { date: "2024-02-05", avgScore: 65, passing: 33 },
  { date: "2024-02-12", avgScore: 67, passing: 35 },
  { date: "2024-02-19", avgScore: 66, passing: 34 },
  { date: "2024-02-26", avgScore: 69, passing: 36 },
  { date: "2024-03-04", avgScore: 68, passing: 35 },
  { date: "2024-03-11", avgScore: 71, passing: 37 },
  { date: "2024-03-18", avgScore: 70, passing: 38 },
  { date: "2024-03-25", avgScore: 73, passing: 40 },
  { date: "2024-04-01", avgScore: 72, passing: 39 },
  { date: "2024-04-08", avgScore: 74, passing: 41 },
  { date: "2024-04-15", avgScore: 75, passing: 41 },
  { date: "2024-04-22", avgScore: 76, passing: 42 },
  { date: "2024-04-29", avgScore: 74, passing: 40 },
  { date: "2024-05-06", avgScore: 77, passing: 43 },
  { date: "2024-05-13", avgScore: 79, passing: 44 },
  { date: "2024-05-20", avgScore: 78, passing: 43 },
  { date: "2024-05-27", avgScore: 81, passing: 45 },
  { date: "2024-06-03", avgScore: 80, passing: 44 },
  { date: "2024-06-10", avgScore: 82, passing: 45 },
  { date: "2024-06-17", avgScore: 84, passing: 46 },
  { date: "2024-06-24", avgScore: 83, passing: 45 },
];

const chartConfig = {
  avgScore: {
    label: "Avg. SQI Score",
    color: "var(--primary)",
  },
  passing: {
    label: "Passing Students",
    color: "var(--primary)",
  },
} satisfies ChartConfig;

export function DashboardChart() {
  const isMobile = useIsMobile();
  const [timeRange, setTimeRange] = React.useState(() =>
    isMobile ? "30d" : "90d"
  );

  const filteredData = chartData.filter((item) => {
    const date = new Date(item.date);
    const referenceDate = new Date("2024-06-30");
    let daysToSubtract = 90;
    if (timeRange === "30d") daysToSubtract = 30;
    else if (timeRange === "7d") daysToSubtract = 7;
    const startDate = new Date(referenceDate);
    startDate.setDate(startDate.getDate() - daysToSubtract);
    return date >= startDate;
  });

  return (
    <Card className="@container/card">
      <CardHeader>
        <CardTitle>Class Performance</CardTitle>
        <CardDescription>
          <span className="hidden @[540px]/card:block">
            Average SQI score trend over time
          </span>
          <span className="@[540px]/card:hidden">Avg. SQI trend</span>
        </CardDescription>
        <CardAction>
          <ToggleGroup
            type="single"
            value={timeRange}
            onValueChange={setTimeRange}
            variant="outline"
            className="hidden *:data-[slot=toggle-group-item]:px-4! @[767px]/card:flex"
          >
            <ToggleGroupItem value="90d">Last 3 months</ToggleGroupItem>
            <ToggleGroupItem value="30d">Last 30 days</ToggleGroupItem>
            <ToggleGroupItem value="7d">Last 7 days</ToggleGroupItem>
          </ToggleGroup>
          <Select value={timeRange} onValueChange={setTimeRange}>
            <SelectTrigger
              className="flex w-40 **:data-[slot=select-value]:block **:data-[slot=select-value]:truncate @[767px]/card:hidden"
              size="sm"
              aria-label="Select a value"
            >
              <SelectValue placeholder="Last 3 months" />
            </SelectTrigger>
            <SelectContent className="rounded-xl">
              <SelectItem value="90d" className="rounded-lg">Last 3 months</SelectItem>
              <SelectItem value="30d" className="rounded-lg">Last 30 days</SelectItem>
              <SelectItem value="7d" className="rounded-lg">Last 7 days</SelectItem>
            </SelectContent>
          </Select>
        </CardAction>
      </CardHeader>
      <CardContent className="px-2 pt-4 sm:px-6 sm:pt-6">
        <ChartContainer
          config={chartConfig}
          className="aspect-auto h-[250px] w-full"
        >
          <AreaChart data={filteredData}>
            <defs>
              <linearGradient id="fillAvgScore" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--color-avgScore)" stopOpacity={0.8} />
                <stop offset="95%" stopColor="var(--color-avgScore)" stopOpacity={0.1} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="date"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              minTickGap={32}
              tickFormatter={(value) =>
                new Date(value).toLocaleDateString("en-US", {
                  month: "short",
                  day: "numeric",
                })
              }
            />
            <ChartTooltip
              cursor={false}
              content={
                <ChartTooltipContent
                  labelFormatter={(value) =>
                    new Date(value).toLocaleDateString("en-US", {
                      month: "short",
                      day: "numeric",
                    })
                  }
                  indicator="dot"
                />
              }
            />
            <Area
              dataKey="avgScore"
              type="natural"
              fill="url(#fillAvgScore)"
              stroke="var(--color-avgScore)"
            />
          </AreaChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}
