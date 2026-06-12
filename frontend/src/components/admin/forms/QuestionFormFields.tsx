import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  SaveIcon,
  XIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react";
import { type CreateQuestionPayload } from "@/services/admin.service";

export type QuestionDraft = CreateQuestionPayload;

export const emptyQuestion = (): QuestionDraft => ({
  question_text: "",
  option_a: "",
  option_b: "",
  option_c: "",
  option_d: "",
  correct_answer: "A",
  marks: 1,
  neg_marks: 0.25,
  importance: "A",
  difficulty: "E",
  type: "Theory",
  expected_time: 1,
  concept_tag: "",
});

type QuestionFormFieldsProps = {
  q: QuestionDraft;
  onChange: (field: keyof QuestionDraft, value: string | number) => void;
  disabled?: boolean;
};

export function QuestionFormFields({
  q,
  onChange,
  disabled,
}: QuestionFormFieldsProps) {
  return (
    <>
      <div className="flex flex-col gap-2">
        <Label>Question Text</Label>
        <Textarea
          value={q.question_text}
          onChange={(e) => onChange("question_text", e.target.value)}
          placeholder="What is 2 + 2?"
          required
          disabled={disabled}
          rows={3}
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        {(["option_a", "option_b", "option_c", "option_d"] as const).map(
          (opt, oi) => (
            <div key={opt} className="flex flex-col gap-1.5">
              <Label>{["A", "B", "C", "D"][oi]}</Label>
              <Textarea
                value={q[opt]}
                onChange={(e) => onChange(opt, e.target.value)}
                placeholder={`Option ${["A", "B", "C", "D"][oi]}`}
                required
                disabled={disabled}
                rows={2}
              />
            </div>
          )
        )}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div className="flex flex-col gap-1.5">
          <Label>Correct Answer</Label>
          <Select
            value={q.correct_answer}
            onValueChange={(v) => onChange("correct_answer", v)}
            disabled={disabled}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {["A", "B", "C", "D"].map((v) => (
                  <SelectItem key={v} value={v}>
                    {v}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Marks</Label>
          <Input
            type="number"
            min={0}
            step={0.25}
            value={q.marks}
            onChange={(e) =>
              onChange("marks", parseFloat(e.target.value) || 0)
            }
            required
            disabled={disabled}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Neg. Marks</Label>
          <Input
            type="number"
            min={0}
            step={0.25}
            value={q.neg_marks}
            onChange={(e) =>
              onChange("neg_marks", parseFloat(e.target.value) || 0)
            }
            required
            disabled={disabled}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Exp. Time (min)</Label>
          <Input
            type="number"
            min={0}
            step={0.1}
            value={q.expected_time}
            onChange={(e) =>
              onChange("expected_time", parseFloat(e.target.value) || 0)
            }
            required
            disabled={disabled}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div className="flex flex-col gap-1.5">
          <Label>Importance</Label>
          <Select
            value={q.importance}
            onValueChange={(v) => onChange("importance", v)}
            disabled={disabled}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="A">A (High)</SelectItem>
                <SelectItem value="B">B (Medium)</SelectItem>
                <SelectItem value="C">C (Low)</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Difficulty</Label>
          <Select
            value={q.difficulty}
            onValueChange={(v) => onChange("difficulty", v)}
            disabled={disabled}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="E">Easy</SelectItem>
                <SelectItem value="M">Medium</SelectItem>
                <SelectItem value="H">Hard</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Type</Label>
          <Select
            value={q.type}
            onValueChange={(v) => onChange("type", v)}
            disabled={disabled}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="Theory">Theory</SelectItem>
                <SelectItem value="Practical">Practical</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Concept Tag</Label>
          <Input
            value={q.concept_tag}
            onChange={(e) => onChange("concept_tag", e.target.value)}
            placeholder="basic_arithmetic"
            disabled={disabled}
          />
        </div>
      </div>
    </>
  );
}

type QuestionFormActionsProps = {
  onSave: () => void;
  onCancel: () => void;
  onAddMore?: () => void;
  onRemove?: () => void;
  saving?: boolean;
  canRemove?: boolean;
  isEditing?: boolean;
};

export function QuestionFormActions({
  onSave,
  onCancel,
  onAddMore,
  onRemove,
  saving,
  canRemove,
  isEditing,
}: QuestionFormActionsProps) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex gap-1">
        {onAddMore && (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={onAddMore}
          >
            <PlusIcon className="size-3 mr-1" /> More
          </Button>
        )}
        {onRemove && canRemove && (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className="text-destructive hover:text-destructive"
            onClick={onRemove}
          >
            <Trash2Icon className="size-3 mr-1" /> Remove
          </Button>
        )}
      </div>
      <div className="flex gap-1">
        <Button size="sm" onClick={onSave} disabled={saving}>
          <SaveIcon className="size-3 mr-1" />{" "}
          {saving ? "Saving\u2026" : isEditing ? "Save" : "Add"}
        </Button>
        <Button size="sm" variant="outline" onClick={onCancel}>
          <XIcon className="size-3 mr-1" /> Cancel
        </Button>
      </div>
    </div>
  );
}
