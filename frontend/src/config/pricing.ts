import type { IntegrityPolicy } from "@/services/types";

export const PRICING = {
  base_rate_per_student: 1,
  timing_flat: 10,
  autosave_flat: 5,
  tab_flat: 3,
  video_rate_per_student_min: 0.05,
};

export function computeEstimatedCost(
  policy: IntegrityPolicy,
  durationMinutes: number,
  studentCount: number
): number {
  if (studentCount <= 0) return 0;
  let cost = PRICING.base_rate_per_student * studentCount;
  if (policy.server_timing) cost += PRICING.timing_flat;
  if (policy.autosave) cost += PRICING.autosave_flat;
  if (policy.tab_switch_detect) cost += PRICING.tab_flat;
  if (policy.video_proctoring)
    cost += PRICING.video_rate_per_student_min * durationMinutes * studentCount;
  return Math.round(cost * 100) / 100;
}
