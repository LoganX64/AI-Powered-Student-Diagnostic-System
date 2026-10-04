import { PRICING } from "@/config/pricing";
import type { IntegrityPolicy } from "@/services/dashboard.service";

type CostSummaryProps = {
  policy: IntegrityPolicy;
  durationMin: number;
  count: number;
};

type Line = { label: string; amount: number };

/**
 * Itemised assignment cost. Shared by the Assign Test tab and the per-student
 * dialog so a quoted total is identical wherever the assignment is created.
 */
export function CostSummary({ policy, durationMin, count }: CostSummaryProps) {
  const lines: Line[] = [
    { label: `Base (${count} × $${PRICING.base_rate_per_student})`, amount: PRICING.base_rate_per_student * count },
  ];

  if (policy.server_timing) lines.push({ label: "Timing", amount: PRICING.timing_flat });
  if (policy.autosave) lines.push({ label: "Autosave", amount: PRICING.autosave_flat });
  if (policy.tab_switch_detect) lines.push({ label: "Tab detect", amount: PRICING.tab_flat });
  if (policy.video_proctoring) {
    lines.push({
      label: "Video",
      amount: PRICING.video_rate_per_student_min * durationMin * count,
    });
  }

  const total = lines.reduce((sum, l) => sum + l.amount, 0);

  return (
    <div className="rounded-md border text-sm">
      {lines.map((l) => (
        <div key={l.label} className="flex justify-between px-3 py-1.5">
          <span className="text-muted-foreground">{l.label}</span>
          <span>
            {l.label.startsWith("Base") ? "" : "+"}${l.amount.toFixed(2)}
          </span>
        </div>
      ))}
      <div className="flex justify-between border-t px-3 py-2 font-semibold">
        <span>Total</span>
        <span>${total.toFixed(2)}</span>
      </div>
    </div>
  );
}