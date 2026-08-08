import { useState } from "react";
import { toast } from "sonner";
import { PlusIcon, Trash2Icon } from "lucide-react";
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
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
    if (!id || id < 1) {
      toast.error("Please enter a valid Test ID");
      return;
    }

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
      <CardHeader>
        <CardTitle>Add Questions to Test</CardTitle>
        <CardDescription>
          Add one or more questions to an existing test.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-6">
          {/* Test ID */}
          <div className="flex flex-col gap-2">
            <Label htmlFor="q-test-id">Test ID</Label>
            <Input
              id="q-test-id"
              type="number"
              min={1}
              placeholder="1"
              value={testId}
              onChange={(e) => setTestId(e.target.value)}
              readOnly={!!testIdProp}
              required
              className="max-w-[180px]"
            />
          </div>

          <Separator />

          {/* Question list */}
          <div className="flex flex-col gap-6">
            {questions.map((q, idx) => (
              <div key={idx} className="flex flex-col gap-4 rounded-lg border p-4">
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
                <div className="flex flex-col gap-2">
                  <Label>Question Text</Label>
                  <Input
                    value={q.question_text}
                    onChange={(e) => update(idx, "question_text", e.target.value)}
                    placeholder="What is 2 + 2?"
                    required
                  />
                </div>

                {/* Options */}
                <div className="grid grid-cols-2 gap-3">
                  {(["option_a", "option_b", "option_c", "option_d"] as const).map((opt, oi) => (
                    <div key={opt} className="flex flex-col gap-1.5">
                      <Label>{["A", "B", "C", "D"][oi]}</Label>
                      <Input
                        value={q[opt]}
                        onChange={(e) => update(idx, opt, e.target.value)}
                        placeholder={`Option ${["A", "B", "C", "D"][oi]}`}
                        required
                      />
                    </div>
                  ))}
                </div>

                {/* Correct answer + Marks */}
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <div className="flex flex-col gap-1.5">
                    <Label>Correct Answer</Label>
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
                  <div className="flex flex-col gap-1.5">
                    <Label>Marks</Label>
                    <Input
                      type="number"
                      min={0}
                      step={0.25}
                      value={q.marks}
                      onChange={(e) => update(idx, "marks", parseFloat(e.target.value) || 0)}
                      required
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label>Neg. Marks</Label>
                    <Input
                      type="number"
                      min={0}
                      step={0.25}
                      value={q.neg_marks}
                      onChange={(e) => update(idx, "neg_marks", parseFloat(e.target.value) || 0)}
                      required
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label>Exp. Time (min)</Label>
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

                {/* Importance / Difficulty / Type / Concept Tag */}
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <div className="flex flex-col gap-1.5">
                    <Label>Importance</Label>
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
                  <div className="flex flex-col gap-1.5">
                    <Label>Difficulty</Label>
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
                  <div className="flex flex-col gap-1.5">
                    <Label>Type</Label>
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
                  <div className="flex flex-col gap-1.5">
                    <Label>Concept Tag</Label>
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
            className="w-fit gap-2"
            onClick={addQuestion}
          >
            <PlusIcon className="size-4" />
            Add Another Question
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
