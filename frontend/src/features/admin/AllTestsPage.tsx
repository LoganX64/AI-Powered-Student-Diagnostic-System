import { useEffect, useState, useCallback } from "react";
import {
  SearchIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronDownIcon,
  ChevronRightIcon as ChevronRightSmallIcon,
  PlusIcon,
  PencilIcon,
  SaveIcon,
  XIcon,
} from "lucide-react";
import { toast } from "sonner";
import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiFetch } from "@/lib/api";
import {
  getTests,
  createQuestions,
  updateQuestion,
  type Test,
  type CreateQuestionPayload,
  type PaginatedResponse,
} from "@/services/admin.service";

const PAGE_SIZE = 50;

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
  importance: "A",
  difficulty: "E",
  type: "Theory",
  expected_time: 1,
  concept_tag: "",
});

function QuestionFormFields({
  q,
  onChange,
}: {
  q: QuestionDraft;
  onChange: (field: keyof QuestionDraft, value: string | number) => void;
}) {
  return (
    <>
      <Input
        value={q.question_text}
        onChange={(e) => onChange("question_text", e.target.value)}
        placeholder="Question text"
        required
      />
      <div className="grid grid-cols-2 gap-2">
        <Input value={q.option_a} onChange={(e) => onChange("option_a", e.target.value)} placeholder="Option A" required />
        <Input value={q.option_b} onChange={(e) => onChange("option_b", e.target.value)} placeholder="Option B" required />
        <Input value={q.option_c} onChange={(e) => onChange("option_c", e.target.value)} placeholder="Option C" required />
        <Input value={q.option_d} onChange={(e) => onChange("option_d", e.target.value)} placeholder="Option D" required />
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <div className="flex flex-col gap-1">
          <Label className="text-xs">Correct Answer</Label>
          <select className="border rounded px-2 py-1 text-sm bg-background" value={q.correct_answer} onChange={(e) => onChange("correct_answer", e.target.value)}>
            <option value="A">A</option><option value="B">B</option><option value="C">C</option><option value="D">D</option>
          </select>
        </div>
        <div className="flex flex-col gap-1">
          <Label className="text-xs">Marks</Label>
          <Input type="number" min={0} step={0.25} value={q.marks} onChange={(e) => onChange("marks", parseFloat(e.target.value) || 0)} />
        </div>
        <div className="flex flex-col gap-1">
          <Label className="text-xs">Neg. Marks</Label>
          <Input type="number" min={0} step={0.25} value={q.neg_marks} onChange={(e) => onChange("neg_marks", parseFloat(e.target.value) || 0)} />
        </div>
        <div className="flex flex-col gap-1">
          <Label className="text-xs">Exp. Time (min)</Label>
          <Input type="number" min={0} step={0.1} value={q.expected_time} onChange={(e) => onChange("expected_time", parseFloat(e.target.value) || 0)} />
        </div>
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <div className="flex flex-col gap-1">
          <Label className="text-xs">Importance</Label>
          <select className="border rounded px-2 py-1 text-sm bg-background" value={q.importance} onChange={(e) => onChange("importance", e.target.value)}>
            <option value="A">A (High)</option><option value="B">B (Medium)</option><option value="C">C (Low)</option>
          </select>
        </div>
        <div className="flex flex-col gap-1">
          <Label className="text-xs">Difficulty</Label>
          <select className="border rounded px-2 py-1 text-sm bg-background" value={q.difficulty} onChange={(e) => onChange("difficulty", e.target.value)}>
            <option value="E">Easy</option><option value="M">Medium</option><option value="H">Hard</option>
          </select>
        </div>
        <div className="flex flex-col gap-1">
          <Label className="text-xs">Type</Label>
          <select className="border rounded px-2 py-1 text-sm bg-background" value={q.type} onChange={(e) => onChange("type", e.target.value)}>
            <option value="Theory">Theory</option><option value="Practical">Practical</option>
          </select>
        </div>
        <div className="flex flex-col gap-1">
          <Label className="text-xs">Concept Tag</Label>
          <Input value={q.concept_tag} onChange={(e) => onChange("concept_tag", e.target.value)} placeholder="basic_arithmetic" />
        </div>
      </div>
    </>
  );
}

export function AllTestsPage() {
  const [tests, setTests] = useState<Test[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");

  const [expandedTestId, setExpandedTestId] = useState<number | null>(null);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [loadingQuestions, setLoadingQuestions] = useState(false);

  const [showAddForm, setShowAddForm] = useState(false);
  const [newQuestions, setNewQuestions] = useState<QuestionDraft[]>([emptyQuestion()]);
  const [addingQuestions, setAddingQuestions] = useState(false);

  const [editingQuestionId, setEditingQuestionId] = useState<number | null>(null);
  const [questionForm, setQuestionForm] = useState<Partial<CreateQuestionPayload>>({});
  const [savingQuestion, setSavingQuestion] = useState(false);

  const fetchTests = useCallback(async (off: number, searchTerm: string) => {
    try {
      const res = await getTests({ limit: PAGE_SIZE, offset: off, search: searchTerm || undefined });
      setTests(res.data ?? []);
      setTotal(res.total);
    } catch {}
  }, []);

  useEffect(() => {
    fetchTests(offset, search);
  }, [offset, search, fetchTests]);

  const fetchQuestions = useCallback(async (testId: number) => {
    setLoadingQuestions(true);
    try {
      const res = await apiFetch<PaginatedResponse<Question>>(`/admin/tests/${testId}/questions?limit=100&offset=0`);
      setQuestions(res.data ?? []);
    } catch {
      setQuestions([]);
    } finally {
      setLoadingQuestions(false);
    }
  }, []);

  const handleExpand = (testId: number) => {
    if (expandedTestId === testId) {
      setExpandedTestId(null);
      setQuestions([]);
      setShowAddForm(false);
      setEditingQuestionId(null);
    } else {
      setExpandedTestId(testId);
      setShowAddForm(false);
      setEditingQuestionId(null);
      fetchQuestions(testId);
    }
  };

  const updateNewQuestion = (index: number, field: keyof QuestionDraft, value: string | number) => {
    setNewQuestions((prev) => prev.map((q, i) => (i === index ? { ...q, [field]: value } : q)));
  };

  const handleSubmitNewQuestions = async () => {
    if (!expandedTestId) return;
    try {
      setAddingQuestions(true);
      const res = await createQuestions(expandedTestId, newQuestions);
      toast.success(`${res.count} question(s) added`);
      setNewQuestions([emptyQuestion()]);
      setShowAddForm(false);
      fetchQuestions(expandedTestId);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setAddingQuestions(false);
    }
  };

  const startEditQuestion = (q: Question) => {
    setEditingQuestionId(q.id);
    setQuestionForm({
      question_text: q.question_text,
      option_a: q.option_a,
      option_b: q.option_b,
      option_c: q.option_c,
      option_d: q.option_d,
      correct_answer: q.correct_answer as "A" | "B" | "C" | "D",
      marks: q.marks,
      neg_marks: q.neg_marks,
      importance: q.importance,
      difficulty: q.difficulty,
      type: q.type,
      expected_time: q.expected_time,
      concept_tag: q.concept_tag,
    });
  };

  const handleSaveQuestion = async (questionId: number) => {
    if (!expandedTestId) return;
    try {
      setSavingQuestion(true);
      await updateQuestion(expandedTestId, questionId, questionForm as CreateQuestionPayload);
      toast.success("Question updated");
      setEditingQuestionId(null);
      fetchQuestions(expandedTestId);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSavingQuestion(false);
    }
  };

  const handleSearch = () => {
    setOffset(0);
    setSearch(searchInput);
  };

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <SiteHeader title="All Tests" />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">
          <div className="flex items-center gap-3">
            <div className="relative flex-1 max-w-sm">
              <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
              <Input
                placeholder="Search by test title..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                className="pl-9"
              />
            </div>
            <Button variant="outline" onClick={handleSearch}>Search</Button>
          </div>

          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">All Tests</h2>
              <Badge variant="secondary">{total}</Badge>
            </div>

            {tests.length === 0 ? (
              <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                <p className="text-sm text-muted-foreground">No tests found.</p>
              </div>
            ) : (
              <div className="flex flex-col gap-2">
                {tests.map((test) => {
                  const isExpanded = expandedTestId === test.test_id;
                  return (
                    <div key={test.test_id} className="rounded-lg border overflow-hidden">
                      <div
                        className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-muted/50"
                        onClick={() => handleExpand(test.test_id)}
                      >
                        {isExpanded ? (
                          <ChevronDownIcon className="size-4 text-muted-foreground shrink-0" />
                        ) : (
                          <ChevronRightSmallIcon className="size-4 text-muted-foreground shrink-0" />
                        )}
                        <span className="font-medium flex-1">{test.title}</span>
                        <Badge variant="secondary" className="hidden sm:inline-flex">{test.subject_name || `#${test.subject_id}`}</Badge>
                        <Badge variant="outline" className="hidden sm:inline-flex">{test.coach_name || `#${test.coach_id}`}</Badge>
                        <span className="text-sm text-muted-foreground">{test.duration}m</span>
                      </div>

                      {isExpanded && (
                        <div className="border-t bg-muted/20 p-4 flex flex-col gap-4">
                          {loadingQuestions ? (
                            <p className="text-sm text-muted-foreground">Loading questions...</p>
                          ) : questions.length === 0 && !showAddForm ? (
                            <div className="flex flex-col items-center gap-3 py-4">
                              <p className="text-sm text-muted-foreground">No questions yet.</p>
                              <Button size="sm" onClick={() => { setShowAddForm(true); setNewQuestions([emptyQuestion()]); }}>
                                <PlusIcon className="size-4 mr-1" /> Add Questions
                              </Button>
                            </div>
                          ) : null}

                          {showAddForm && (
                            <div className="flex flex-col gap-4 rounded-lg border p-4">
                              <div className="flex items-center justify-between">
                                <h4 className="text-sm font-semibold">Add Questions to: {test.title}</h4>
                                <Button variant="ghost" size="sm" onClick={() => setShowAddForm(false)}><XIcon className="size-4" /></Button>
                              </div>
                              {newQuestions.map((q, idx) => (
                                <div key={idx} className="flex flex-col gap-3 rounded-md border p-3">
                                  <div className="flex items-center justify-between">
                                    <span className="text-xs font-semibold text-muted-foreground">Question {idx + 1}</span>
                                    {newQuestions.length > 1 && (
                                      <Button type="button" variant="ghost" size="icon" className="size-6 text-destructive" onClick={() => setNewQuestions((prev) => prev.filter((_, i) => i !== idx))}>
                                        <XIcon className="size-3" />
                                      </Button>
                                    )}
                                  </div>
                                  <QuestionFormFields q={q} onChange={(field, value) => updateNewQuestion(idx, field, value)} />
                                </div>
                              ))}
                              <div className="flex gap-2">
                                <Button size="sm" onClick={handleSubmitNewQuestions} disabled={addingQuestions}>
                                  {addingQuestions ? "Adding\u2026" : `Add ${newQuestions.length} Question(s)`}
                                </Button>
                                <Button size="sm" variant="outline" onClick={() => setNewQuestions((prev) => [...prev, emptyQuestion()])}>
                                  <PlusIcon className="size-3 mr-1" /> More
                                </Button>
                              </div>
                            </div>
                          )}

                          {editingQuestionId && (
                            <div className="flex flex-col gap-3 rounded-lg border p-4">
                              <div className="flex items-center justify-between">
                                <h4 className="text-sm font-semibold">Edit Question</h4>
                                <div className="flex gap-1">
                                  <Button size="sm" onClick={() => handleSaveQuestion(editingQuestionId)} disabled={savingQuestion}>
                                    <SaveIcon className="size-3 mr-1" /> {savingQuestion ? "Saving\u2026" : "Save"}
                                  </Button>
                                  <Button size="sm" variant="outline" onClick={() => setEditingQuestionId(null)}>
                                    <XIcon className="size-3 mr-1" /> Cancel
                                  </Button>
                                </div>
                              </div>
                              <QuestionFormFields
                                q={questionForm as QuestionDraft}
                                onChange={(field, value) => setQuestionForm((prev) => ({ ...prev, [field]: value }))}
                              />
                            </div>
                          )}

                          {questions.length > 0 && (
                            <>
                              <div className="flex items-center justify-between">
                                <h4 className="text-sm font-semibold">Questions ({questions.length})</h4>
                                <Button size="sm" variant="outline" onClick={() => { setShowAddForm(true); setNewQuestions([emptyQuestion()]); }}>
                                  <PlusIcon className="size-3 mr-1" /> Add
                                </Button>
                              </div>
                              <div className="rounded-md border overflow-x-auto">
                                <Table>
                                  <TableHeader>
                                    <TableRow>
                                      <TableHead className="w-10">#</TableHead>
                                      <TableHead>Question</TableHead>
                                      <TableHead className="w-12">A</TableHead>
                                      <TableHead className="w-12">B</TableHead>
                                      <TableHead className="w-12">C</TableHead>
                                      <TableHead className="w-12">D</TableHead>
                                      <TableHead className="w-12">Ans</TableHead>
                                      <TableHead className="w-14">Marks</TableHead>
                                      <TableHead className="w-16">Diff</TableHead>
                                      <TableHead className="w-10"></TableHead>
                                    </TableRow>
                                  </TableHeader>
                                  <TableBody>
                                    {questions.map((q, idx) => (
                                      <TableRow key={q.id}>
                                        <TableCell className="font-mono text-xs text-muted-foreground">{idx + 1}</TableCell>
                                        <TableCell className="font-medium max-w-[200px] truncate text-sm">{q.question_text}</TableCell>
                                        <TableCell className="text-xs">{q.option_a}</TableCell>
                                        <TableCell className="text-xs">{q.option_b}</TableCell>
                                        <TableCell className="text-xs">{q.option_c}</TableCell>
                                        <TableCell className="text-xs">{q.option_d}</TableCell>
                                        <TableCell><Badge variant="default" className="text-xs">{q.correct_answer}</Badge></TableCell>
                                        <TableCell className="text-muted-foreground text-xs">{q.marks}</TableCell>
                                        <TableCell>
                                          <Badge variant={q.difficulty === "E" ? "secondary" : q.difficulty === "M" ? "outline" : "destructive"} className="text-xs">
                                            {q.difficulty === "E" ? "Easy" : q.difficulty === "M" ? "Med" : "Hard"}
                                          </Badge>
                                        </TableCell>
                                        <TableCell>
                                          <Button variant="ghost" size="icon" className="size-7" onClick={() => startEditQuestion(q)}><PencilIcon className="size-3" /></Button>
                                        </TableCell>
                                      </TableRow>
                                    ))}
                                  </TableBody>
                                </Table>
                              </div>
                            </>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}

            {total > PAGE_SIZE && (
              <div className="flex items-center justify-between pt-2">
                <p className="text-sm text-muted-foreground">
                  Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
                </p>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" disabled={offset === 0} onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}>
                    <ChevronLeftIcon className="size-4" /> Prev
                  </Button>
                  <Button variant="outline" size="sm" disabled={offset + PAGE_SIZE >= total} onClick={() => setOffset((o) => o + PAGE_SIZE)}>
                    Next <ChevronRightIcon className="size-4" />
                  </Button>
                </div>
              </div>
            )}
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
