import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeftIcon, ChevronLeftIcon, ChevronRightIcon, PencilIcon, SaveIcon, XIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { toast } from "sonner";
import { AppSidebar } from "@/components/admin/app-sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
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
  getAssignments,
  updateTest,
  updateQuestion,
  deleteTest,
  type Assignment,
  type CreateTestPayload,
  type CreateQuestionPayload,
  type PaginatedResponse,
} from "@/services/admin.service";
import {
  QuestionFormFields,
  type QuestionDraft,
} from "@/components/admin/forms/QuestionFormFields";

type TestDetail = {
  test_id: number;
  title: string;
  subject_id: number;
  coach_id: number;
  duration: number;
  created_at: string;
  subject_name: string;
  coach_name: string;
};

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

const PAGE_SIZE = 50;

export function TestDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const testId = Number(id);

  const [test, setTest] = useState<TestDetail | null>(null);
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [assignmentTotal, setAssignmentTotal] = useState(0);
  const [assignmentOffset, setAssignmentOffset] = useState(0);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [questionTotal, setQuestionTotal] = useState(0);
  const [questionOffset, setQuestionOffset] = useState(0);

  const [editingTest, setEditingTest] = useState(false);
  const [testForm, setTestForm] = useState({ title: "", subject_id: 0, coach_id: 0, duration: 0 });
  const [savingTest, setSavingTest] = useState(false);

  const [editingQuestionId, setEditingQuestionId] = useState<number | null>(null);
  const [questionForm, setQuestionForm] = useState<Partial<CreateQuestionPayload>>({});
  const [savingQuestion, setSavingQuestion] = useState(false);

  useEffect(() => {
    if (!id) return;
    apiFetch<TestDetail>(`/admin/tests/${id}`).then((data) => {
      setTest(data);
      setTestForm({ title: data.title, subject_id: data.subject_id, coach_id: data.coach_id, duration: data.duration });
    }).catch(() => {});
    if (window.location.search.includes("edit=true")) {
      setEditingTest(true);
    }
  }, [id]);

  const fetchAssignments = useCallback(async (off: number) => {
    if (!id) return;
    try {
      const res = await getAssignments({ limit: PAGE_SIZE, offset: off, test_id: Number(id) });
      setAssignments(res.data ?? []);
      setAssignmentTotal(res.total);
    } catch {
      // silently ignore
    }
  }, [id]);

  const fetchQuestions = useCallback(async (off: number) => {
    if (!id) return;
    try {
      const res = await apiFetch<PaginatedResponse<Question>>(`/admin/tests/${id}/questions?limit=${PAGE_SIZE}&offset=${off}`);
      setQuestions(res.data ?? []);
      setQuestionTotal(res.total);
    } catch {
      // silently ignore
    }
  }, [id]);

  useEffect(() => {
    fetchAssignments(assignmentOffset);
  }, [assignmentOffset, fetchAssignments]);

  useEffect(() => {
    fetchQuestions(questionOffset);
  }, [questionOffset, fetchQuestions]);

  const handleSaveTest = async () => {
    if (!testForm.title || !testForm.duration) {
      toast.error("Title and duration are required");
      return;
    }
    try {
      setSavingTest(true);
      await updateTest(testId, {
        title: testForm.title,
        subject_id: test.subject_id,
        coach_id: test.coach_id,
        duration: testForm.duration,
      });
      toast.success("Test updated");
      setEditingTest(false);
      const data = await apiFetch<TestDetail>(`/admin/tests/${id}`);
      setTest(data);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSavingTest(false);
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
    try {
      setSavingQuestion(true);
      await updateQuestion(testId, questionId, questionForm as CreateQuestionPayload);
      toast.success("Question updated");
      setEditingQuestionId(null);
      fetchQuestions(questionOffset);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSavingQuestion(false);
    }
  };

  const handleDeleteTest = async () => {
    try {
      await deleteTest(testId);
      toast.success(`Test "${test.title}" deleted`);
      navigate("/admin/all-tests");
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  if (!test) {
    return (
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset>
          <SiteHeader title="Test Detail" />
          <div className="flex flex-1 items-center justify-center p-6">
            <p className="text-muted-foreground">Loading...</p>
          </div>
        </SidebarInset>
      </SidebarProvider>
    );
  }

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <SiteHeader title={test.title} />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

          {/* Back button + test info */}
          <div className="flex flex-col gap-4">
            <Button
              variant="ghost"
              size="sm"
              className="w-fit"
              onClick={() => navigate("/admin/all-tests")}
            >
              <ArrowLeftIcon className="size-4 mr-2" /> Back to All Tests
            </Button>

            {editingTest ? (
              <div className="flex flex-col gap-3 rounded-lg border p-4">
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="flex flex-col gap-1">
                    <Label htmlFor="edit-title">Title</Label>
                    <Input
                      id="edit-title"
                      value={testForm.title}
                      onChange={(e) => setTestForm({ ...testForm, title: e.target.value })}
                    />
                  </div>
                  <div className="flex flex-col gap-1">
                    <Label htmlFor="edit-duration">Duration (minutes)</Label>
                    <Input
                      id="edit-duration"
                      type="number"
                      value={testForm.duration}
                      onChange={(e) => setTestForm({ ...testForm, duration: Number(e.target.value) })}
                    />
                  </div>
                </div>
                <div className="flex gap-2">
                  <Button size="sm" onClick={handleSaveTest} disabled={savingTest}>
                    <SaveIcon className="size-4 mr-1" /> {savingTest ? "Saving…" : "Save"}
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => setEditingTest(false)}>
                    <XIcon className="size-4 mr-1" /> Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <div className="flex items-center gap-4 flex-wrap">
                <h2 className="text-lg font-semibold">{test.title}</h2>
                <Badge variant="outline">ID: {test.test_id}</Badge>
                <Badge variant="secondary">Duration: {test.duration}s</Badge>
                <Badge variant="secondary">Subject: {test.subject_name || `#${test.subject_id}`}</Badge>
                <Badge variant="secondary">Coach: {test.coach_name || `#${test.coach_id}`}</Badge>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setEditingTest(true)}
                >
                  <PencilIcon className="size-4 mr-1" /> Edit
                </Button>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-destructive hover:bg-destructive hover:text-destructive-foreground"
                    >
                      <Trash2Icon className="size-4 mr-1" /> Delete
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Delete Test</AlertDialogTitle>
                      <AlertDialogDescription>
                        Are you sure you want to delete{" "}
                        <span className="font-semibold">{test.title}</span>?
                        This will also delete all related questions and assignments.
                        This action cannot be undone.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction
                        onClick={handleDeleteTest}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                      >
                        Delete
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            )}
          </div>

          {/* Tabs */}
          <Tabs defaultValue="assignments" className="w-full">
            <TabsList className="mb-2">
              <TabsTrigger value="assignments">
                Assigned Students ({assignmentTotal})
              </TabsTrigger>
              <TabsTrigger value="questions">
                Questions ({questionTotal})
              </TabsTrigger>
            </TabsList>

            {/* Assignments tab */}
            <TabsContent value="assignments" className="flex flex-col gap-3">
              {assignments.length === 0 ? (
                <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                  <p className="text-sm text-muted-foreground">No students assigned to this test.</p>
                </div>
              ) : (
                <>
                  <div className="rounded-lg border overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-16">ID</TableHead>
                          <TableHead>Student Name</TableHead>
                          <TableHead>Student Code</TableHead>
                          <TableHead className="w-24">Status</TableHead>
                          <TableHead>Assigned At</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {assignments.map((a) => (
                          <TableRow key={a.id}>
                            <TableCell className="font-mono text-sm text-muted-foreground">{a.id}</TableCell>
                            <TableCell className="font-medium">{a.student_name}</TableCell>
                            <TableCell>
                              <Badge variant="outline" className="font-mono">{a.student_code}</Badge>
                            </TableCell>
                            <TableCell>
                              <Badge variant={a.status === "assigned" ? "secondary" : "default"}>{a.status}</Badge>
                            </TableCell>
                            <TableCell className="text-muted-foreground text-sm">{a.assigned_at}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>

                  {assignmentTotal > PAGE_SIZE && (
                    <div className="flex items-center justify-between pt-2">
                      <p className="text-sm text-muted-foreground">
                        Showing {assignmentOffset + 1}–{Math.min(assignmentOffset + PAGE_SIZE, assignmentTotal)} of {assignmentTotal}
                      </p>
                      <div className="flex gap-2">
                        <Button variant="outline" size="sm" disabled={assignmentOffset === 0} onClick={() => setAssignmentOffset((o) => Math.max(0, o - PAGE_SIZE))}>
                          <ChevronLeftIcon className="size-4" /> Prev
                        </Button>
                        <Button variant="outline" size="sm" disabled={assignmentOffset + PAGE_SIZE >= assignmentTotal} onClick={() => setAssignmentOffset((o) => o + PAGE_SIZE)}>
                          Next <ChevronRightIcon className="size-4" />
                        </Button>
                      </div>
                    </div>
                  )}
                </>
              )}
            </TabsContent>

            {/* Questions tab */}
            <TabsContent value="questions" className="flex flex-col gap-3">
              {questions.length === 0 ? (
                <div className="flex flex-col h-32 items-center justify-center rounded-lg border border-dashed gap-3">
                  <p className="text-sm text-muted-foreground">No questions in this test.</p>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate("/admin/tests")}
                  >
                    <PlusIcon className="size-4 mr-1" /> Add Questions
                  </Button>
                </div>
              ) : (
                <>
                  {/* Edit question panel */}
                  {editingQuestionId && (
                    <div className="flex flex-col gap-3 rounded-lg border p-4">
                      <div className="flex items-center justify-between">
                        <h4 className="text-sm font-semibold">Edit Question</h4>
                      </div>
                      <QuestionFormFields
                        q={questionForm as QuestionDraft}
                        onChange={(field, value) => setQuestionForm((prev) => ({ ...prev, [field]: value }))}
                      />
                      <div className="flex justify-end gap-2">
                        <Button size="sm" onClick={() => handleSaveQuestion(editingQuestionId)} disabled={savingQuestion}>
                          <SaveIcon className="size-3 mr-1" /> {savingQuestion ? "Saving\u2026" : "Save"}
                        </Button>
                        <Button size="sm" variant="outline" onClick={() => setEditingQuestionId(null)}>
                          Cancel
                        </Button>
                      </div>
                    </div>
                  )}

                  <div className="rounded-lg border overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-16">#</TableHead>
                          <TableHead>Question</TableHead>
                          <TableHead className="w-16">A</TableHead>
                          <TableHead className="w-16">B</TableHead>
                          <TableHead className="w-16">C</TableHead>
                          <TableHead className="w-16">D</TableHead>
                          <TableHead className="w-16">Answer</TableHead>
                          <TableHead className="w-16">Marks</TableHead>
                          <TableHead className="w-16">Neg</TableHead>
                          <TableHead className="w-20">Difficulty</TableHead>
                          <TableHead className="w-16">Import</TableHead>
                          <TableHead className="w-16">Type</TableHead>
                          <TableHead className="w-16">Time</TableHead>
                          <TableHead className="w-24">Tag</TableHead>
                          <TableHead className="w-16"></TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {questions.map((q, idx) => (
                          <TableRow key={q.id}>
                            <TableCell className="font-mono text-sm text-muted-foreground">
                              {questionOffset + idx + 1}
                            </TableCell>
                            <TableCell className="font-medium max-w-md truncate">{q.question_text}</TableCell>
                            <TableCell className="text-sm">{q.option_a}</TableCell>
                            <TableCell className="text-sm">{q.option_b}</TableCell>
                            <TableCell className="text-sm">{q.option_c}</TableCell>
                            <TableCell className="text-sm">{q.option_d}</TableCell>
                            <TableCell>
                              <Badge variant="default">{q.correct_answer}</Badge>
                            </TableCell>
                            <TableCell className="text-muted-foreground">{q.marks}</TableCell>
                            <TableCell className="text-muted-foreground">{q.neg_marks}</TableCell>
                            <TableCell>
                              <Badge variant={
                                q.difficulty === "E" ? "secondary" :
                                q.difficulty === "M" ? "outline" : "destructive"
                              }>
                                {q.difficulty === "E" ? "Easy" : q.difficulty === "M" ? "Medium" : "Hard"}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <Badge variant={q.importance === "A" ? "default" : q.importance === "B" ? "secondary" : "outline"}>
                                {q.importance}
                              </Badge>
                            </TableCell>
                            <TableCell className="text-sm">{q.type}</TableCell>
                            <TableCell className="text-sm">{q.expected_time}m</TableCell>
                            <TableCell className="text-sm truncate max-w-[120px]" title={q.concept_tag}>{q.concept_tag}</TableCell>
                            <TableCell>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="size-7"
                                onClick={() => startEditQuestion(q)}
                              >
                                <PencilIcon className="size-3" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>

                  {questionTotal > PAGE_SIZE && (
                    <div className="flex items-center justify-between pt-2">
                      <p className="text-sm text-muted-foreground">
                        Showing {questionOffset + 1}–{Math.min(questionOffset + PAGE_SIZE, questionTotal)} of {questionTotal}
                      </p>
                      <div className="flex gap-2">
                        <Button variant="outline" size="sm" disabled={questionOffset === 0} onClick={() => setQuestionOffset((o) => Math.max(0, o - PAGE_SIZE))}>
                          <ChevronLeftIcon className="size-4" /> Prev
                        </Button>
                        <Button variant="outline" size="sm" disabled={questionOffset + PAGE_SIZE >= questionTotal} onClick={() => setQuestionOffset((o) => o + PAGE_SIZE)}>
                          Next <ChevronRightIcon className="size-4" />
                        </Button>
                      </div>
                    </div>
                  )}
                </>
              )}
            </TabsContent>
          </Tabs>

        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
