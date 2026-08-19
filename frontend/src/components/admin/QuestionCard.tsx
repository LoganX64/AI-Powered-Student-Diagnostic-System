import { PencilIcon, Trash2Icon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

type Question = {
  id: number;
  question_text: string;
  option_a: string;
  option_b: string;
  option_c: string;
  option_d: string;
  correct_answer: string;
  marks: number;
  neg_marks: number;
  importance: string;
  difficulty: string;
  type: string;
  expected_time: number;
  concept_tag: string;
};

type QuestionCardProps = {
  index: number;
  question: Question;
  onEdit: () => void;
  onDelete?: () => void;
};

export function QuestionCard({ index, question: q, onEdit, onDelete }: QuestionCardProps) {
  const options = [
    { label: "A", value: q.option_a },
    { label: "B", value: q.option_b },
    { label: "C", value: q.option_c },
    { label: "D", value: q.option_d },
  ];

  return (
    <div className="rounded-lg border p-4 flex flex-col gap-3">
      {/* Header: Question number + Edit/Delete buttons */}
      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold text-muted-foreground">
          Q{index}
        </span>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={onEdit}
          >
            <PencilIcon className="size-3" />
          </Button>
          {onDelete && (
            <Button
              variant="ghost"
              size="icon"
              className="size-7 text-destructive hover:text-destructive"
              onClick={onDelete}
            >
              <Trash2Icon className="size-3" />
            </Button>
          )}
        </div>
      </div>

      {/* Question text - wraps naturally */}
      <p className="text-sm font-medium leading-relaxed">
        {q.question_text}
      </p>

      {/* Options grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
        {options.map((opt) => (
          <div
            key={opt.label}
            className={`flex items-start gap-2 rounded-md border p-2 text-sm ${
              q.correct_answer === opt.label
                ? "border-green-500 bg-green-50 "
                : ""
            }`}
          >
            <Badge
              variant={q.correct_answer === opt.label ? "default" : "secondary"}
              className="shrink-0 size-5 flex items-center justify-center text-xs"
            >
              {opt.label}
            </Badge>
            <span className="leading-relaxed">{opt.value}</span>
          </div>
        ))}
      </div>

      {/* Metadata row */}
      <div className="flex flex-wrap items-center gap-2 pt-1">
        <Badge variant="outline" className="text-xs">
          Marks: {q.marks}
        </Badge>
        <Badge variant="outline" className="text-xs">
          Neg: {q.neg_marks}
        </Badge>
        <Badge
          variant={
            q.difficulty === "E"
              ? "secondary"
              : q.difficulty === "M"
              ? "outline"
              : "destructive"
          }
          className="text-xs"
        >
          {q.difficulty === "E" ? "Easy" : q.difficulty === "M" ? "Medium" : "Hard"}
        </Badge>
        <Badge
          variant={
            q.importance === "high" ? "default" : q.importance === "medium" ? "secondary" : "outline"
          }
          className="text-xs"
        >
          {q.importance === "high" ? "High" : q.importance === "medium" ? "Medium" : "Low"}
        </Badge>
        <Badge variant="secondary" className="text-xs">
          {q.type}
        </Badge>
        <Badge variant="secondary" className="text-xs">
          {q.expected_time}m
        </Badge>
        {q.concept_tag && (
          <Badge variant="outline" className="text-xs ml-auto">
            {q.concept_tag}
          </Badge>
        )}
      </div>
    </div>
  );
}
