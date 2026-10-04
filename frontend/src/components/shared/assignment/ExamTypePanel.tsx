import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import type { IntegrityPolicy } from "@/services/dashboard.service";
import {
  INTEGRITY_TOGGLES,
  PRESETS,
  presetPolicy,
  samePolicy,
} from "./exam-presets";

type ExamTypePanelProps = {
  policy: IntegrityPolicy;
  onChange: (policy: IntegrityPolicy) => void;
};

/**
 * Preset cards plus the individual integrity toggles. Renders inner content
 * only — the caller supplies the surrounding Card so it can sit inline in the
 * Assign Test tab or stacked inside the per-student dialog.
 */
export function ExamTypePanel({ policy, onChange }: ExamTypePanelProps) {
  return (
    <>
      <Label>Exam Type &amp; Integrity</Label>
      <div className="grid gap-2 sm:grid-cols-3">
        {PRESETS.map((p) => {
          const active = samePolicy(policy, presetPolicy(p.key));
          const Icon = p.icon;
          return (
            <button
              type="button"
              key={p.key}
              onClick={() => onChange(presetPolicy(p.key))}
              className={`flex items-start gap-3 rounded-lg border p-3 text-left transition-colors ${
                active
                  ? "border-primary bg-primary/5"
                  : "hover:bg-accent hover:text-accent-foreground"
              }`}
            >
              <Icon className="size-4 mt-0.5 shrink-0" />
              <div>
                <span className="text-sm font-medium">{p.title}</span>
                <span className="block text-xs text-muted-foreground">{p.desc}</span>
              </div>
            </button>
          );
        })}
      </div>

      <div className="grid gap-2 sm:grid-cols-2">
        {INTEGRITY_TOGGLES.map((item) => (
          <div key={item.key} className="flex items-center justify-between rounded-md border px-3 py-2">
            <div>
              <p className="text-sm font-medium">{item.label}</p>
              <p className="text-xs text-muted-foreground">{item.desc}</p>
            </div>
            <Switch
              checked={policy[item.key]}
              onCheckedChange={(v) => onChange({ ...policy, [item.key]: v })}
            />
          </div>
        ))}
      </div>
    </>
  );
}