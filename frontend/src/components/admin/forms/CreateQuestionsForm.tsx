import { useState } from "react";
import { toast } from "sonner";
import { PlusIcon, Trash2Icon } from "lucide-react";
import { createQuestionsBatchSchema, zodErrors } from "@/lib/validations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { createQuestions as adminCreateQuestions, type CreateQuestionPayload } from "@/services/dashboard.service";

type QuestionDraft = CreateQuestionPayload;

const emptyQuestion = (): QuestionDraft => ({
  question_text: "",
  option_a: "",
  option_b: "",
  option_c: "",
  option_d: "",
  correct_answer: "A",
  marks: 1,
  neg_marks: 0.25,
  importance: "high",
  difficulty: "E",
  type: "mcq",
  expected_time: 1,
  concept_tag: "",
});

type Props = {
  testId?: number;
  onCreated?: (testId: number, count: number) => void;
  onSubmit?: (testId: number, data: CreateQuestionPayload[]) => Promise<{ question_ids: number[]; count: number }>;
};

export function CreateQuestionsForm({ testId: testIdProp, onCreated, onSubmit }: Props) {
  const [testId, setTestId] = useState(testIdProp?.toString() ?? "");
  const [questions, setQuestions] = useState<QuestionDraft[]>([emptyQuestion()]);
  const [loading, setLoading] = useState(false);
  const [, setErrors] = useState<Record<string, string>>({});

  const update = (index: number, field: keyof QuestionDraft, value: string | number) => {
    setQuestions((prev) =>
      prev.map((q, i) => (i === index ? { ...q, [field]: value } : q))
    );
  };

  const addQuestion = () => setQuestions((prev) => [...prev, emptyQuestion()]);

  const removeQuestion = (index: number) => {
    if (questions.length === 1) return;
    setQuestions((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = async (e) => {
    e.preventDefault();
    const id = Number(testId);

    const result = createQuestionsBatchSchema.safeParse({ test_id: id, questions });
    if (!result.success) {
      setErrors(zodErrors(result.error));
      return;
    }
    setErrors({});

    try {
      setLoading(true);
      const createFn = onSubmit ?? adminCreateQuestions;
      const res = await createFn(id, questions);
      toast.success(`${res.count} question(s) added to test ${id}`);
      onCreated?.(id, res.count);
      setQuestions([emptyQuestion()]);
      if (!testIdProp) setTestId("");
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader className="pb-0">
        <CardTitle>Add Questions to Test</CardTitle>
        <CardDescription className="flex items-center gap-2">
          Add one or more questions to an existing test.
          {testId && <Badge variant="outline" className="font-mono">Test ID: {testId}</Badge>}
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-4">
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <Separator />

          {/* Question list */}
          <div className="flex flex-col gap-4">
            {questions.map((q, idx) => (
              <div key={idx} className="flex flex-col gap-3 rounded-lg border p-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-muted-foreground">
                    Question {idx + 1}
                  </span>
                  {questions.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="size-7 text-destructive hover:text-destructive"
                      onClick={() => removeQuestion(idx)}
                      aria-label={`Remove question ${idx + 1}`}
                    >
                      <Trash2Icon className="size-4" />
                    </Button>
                  )}
                </div>

                {/* Question text */}
                <div className="flex flex-col gap-1.5">
                  <Label>Question Text</Label>
                  <Textarea
                    rows={2}
                    value={q.question_text}
                    onChange={(e) => {
                      update(idx, "question_text", e.target.value);
                      e.target.style.height = "auto";
                      e.target.style.height = e.target.scrollHeight + "px";
                    }}
                    placeholder="What is 2 + 2?"
                    required
                    className="resize-none overflow-hidden"
                  />
                </div>

                {/* Options — 2×2 grid */}
                <div className="grid grid-cols-2 gap-x-3 gap-y-1.5">
                  {(["option_a", "option_b", "option_c", "option_d"] as const).map((opt, oi) => (
                    <div key={opt} className="flex flex-col gap-1">
                      <Label className="text-xs">{["A", "B", "C", "D"][oi]}</Label>
                      <Input
                        value={q[opt]}
                        onChange={(e) => update(idx, opt, e.target.value)}
                        placeholder={`Option ${["A", "B", "C", "D"][oi]}`}
                        required
                      />
                    </div>
                  ))}
                </div>

                {/* All metadata in a single 4-col row */}
                <div className="grid grid-cols-4 gap-x-3 gap-y-1.5">
                  <div className="flex flex-col gap-1">
                    <Label className="text-xs">Answer</Label>
                    <Select
                      value={q.correct_answer}
                      onValueChange={(v) => update(idx, "correct_answer", v)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          {["A", "B", "C", "D"].map((v) => (
                            <SelectItem key={v} value={v}>{v}</SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="flex flex-col gap-1">
                    <Label className="text-xs">Marks</Label>
                    <Input
                      type="number"
                      min={0}
                      step={0.25}
                      value={q.marks}
                      onChange={(e) => update(idx, "marks", parseFloat(e.target.value) || 0)}
                      required
                    />
                  </div>
                  <div className="flex flex-col gap-1">
                    <Label className="text-xs">Neg Marks</Label>
                    <Input
                      type="number"
                      min={0}
                      step={0.25}
                      value={q.neg_marks}
                      onChange={(e) => update(idx, "neg_marks", parseFloat(e.target.value) || 0)}
                      required
                    />
                  </div>
                  <div className="flex flex-col gap-1">
                    <Label className="text-xs">Time (min)</Label>
                    <Input
                      type="number"
                      min={0}
                      step={0.1}
                      value={q.expected_time}
                      onChange={(e) => update(idx, "expected_time", parseFloat(e.target.value) || 0)}
                      required
                    />
                  </div>
                </div>

                {/* Second metadata row */}
                <div className="grid grid-cols-4 gap-x-3 gap-y-1.5">
                  <div className="flex flex-col gap-1">
                    <Label className="text-xs">Importance</Label>
                    <Select
                      value={q.importance}
                      onValueChange={(v) => update(idx, "importance", v)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectItem value="high">high</SelectItem>
                          <SelectItem value="medium">medium</SelectItem>
                          <SelectItem value="low">low</SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="flex flex-col gap-1">
                    <Label className="text-xs">Difficulty</Label>
                    <Select
                      value={q.difficulty}
                      onValueChange={(v) => update(idx, "difficulty", v)}
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
                  <div className="flex flex-col gap-1">
                    <Label className="text-xs">Type</Label>
                    <Select
                      value={q.type}
                      onValueChange={(v) => update(idx, "type", v)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectItem value="mcq">mcq</SelectItem>
                          <SelectItem value="multi">multi</SelectItem>
                          <SelectItem value="integer">integer</SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="flex flex-col gap-1">
                    <Label className="text-xs">Concept Tag</Label>
                    <Input
                      value={q.concept_tag}
                      onChange={(e) => update(idx, "concept_tag", e.target.value)}
                      placeholder="basic_arithmetic"
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>

          <Button
            type="button"
            variant="outline"
            size="sm"
            className="w-fit gap-1.5"
            onClick={addQuestion}
          >
            <PlusIcon className="size-3.5" />
            Add Question
          </Button>

          <Separator />

          <div className="flex items-center gap-3">
            <Button type="submit" disabled={loading} className="w-fit">
              {loading ? "Submitting…" : `Submit ${questions.length} Question${questions.length > 1 ? "s" : ""}`}
            </Button>
            <span className="text-sm text-muted-foreground">
              {questions.length} question{questions.length > 1 ? "s" : ""} ready
            </span>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
