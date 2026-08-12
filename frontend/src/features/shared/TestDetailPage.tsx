import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeftIcon, ChevronLeftIcon, ChevronRightIcon, PencilIcon, SaveIcon, PlusIcon, Trash2Icon, BarChartIcon } from "lucide-react";
import { toast } from "sonner";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  getAssignments,
  getTest,
  getTestQuestions,
  updateQuestion,
  deleteTest,
  type Assignment,
  type TestDetail,
  type TestQuestion,
  type CreateQuestionPayload,
} from "@/services/dashboard.service";
import {
  QuestionFormFields,
  type QuestionDraft,
} from "@/components/admin/forms/QuestionFormFields";
import { QuestionCard } from "@/components/admin/QuestionCard";
import { EditTestDialog } from "@/components/admin/forms/EditTestDialog";
import { formatDateDDMMYYYY, parseRouteId } from "@/lib/utils";

const PAGE_SIZE = 50;

export function TestDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";
  const isAdmin = role === "admin";
  const testId = parseRouteId(id);

  const [test, setTest] = useState<TestDetail | null>(null);
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [assignmentTotal, setAssignmentTotal] = useState(0);
  const [assignmentOffset, setAssignmentOffset] = useState(0);
  const [questions, setQuestions] = useState<TestQuestion[]>([]);
  const [questionTotal, setQuestionTotal] = useState(0);
  const [questionOffset, setQuestionOffset] = useState(0);

  const [editingTest, setEditingTest] = useState(false);

  const [editingQuestion, setEditingQuestion] = useState<TestQuestion | null>(null);
  const [questionForm, setQuestionForm] = useState<Partial<CreateQuestionPayload>>({});
  const [savingQuestion, setSavingQuestion] = useState(false);

  useEffect(() => {
    if (testId === null) return;
    getTest(testId).then(setTest).catch(() => {});
    if (isAdmin && window.location.search.includes("edit=true")) {
      setEditingTest(true);
    }
  }, [testId, isAdmin]);

  const fetchAssignments = useCallback(async (off: number) => {
    if (testId === null) return;
    try {
      const res = await getAssignments({ limit: PAGE_SIZE, offset: off, test_id: testId });
      setAssignments(res.data ?? []);
      setAssignmentTotal(res.total);
    } catch {
      // silently ignore
    }
  }, [testId]);

  const fetchQuestions = useCallback(async (off: number) => {
    if (testId === null) return;
    try {
      const res = await getTestQuestions(testId, { limit: PAGE_SIZE, offset: off });
      setQuestions(res.data ?? []);
      setQuestionTotal(res.total);
    } catch {
      // silently ignore
    }
  }, [testId]);

  useEffect(() => {
    fetchAssignments(assignmentOffset);
  }, [assignmentOffset, fetchAssignments]);

  useEffect(() => {
    fetchQuestions(questionOffset);
  }, [questionOffset, fetchQuestions]);

  const openEditDialog = (q: TestQuestion) => {
    setEditingQuestion(q);
    setQuestionForm({
      question_text: q.question_text,
      option_a: q.option_a,
      option_b: q.option_b,
      option_c: q.option_c,
      option_d: q.option_d,
      correct_answer: q.correct_answer as "A" | "B" | "C" | "D",
      marks: q.marks,
      neg_marks: q.neg_marks,
      importance: q.importance as "high" | "medium" | "low",
      difficulty: q.difficulty as "E" | "M" | "H",
      type: q.type as "mcq" | "multi" | "integer",
      expected_time: q.expected_time,
      concept_tag: q.concept_tag,
    });
  };

  const handleSaveQuestion = async () => {
    if (!editingQuestion || testId === null) return;
    try {
      setSavingQuestion(true);
      await updateQuestion(testId, editingQuestion.id, questionForm as CreateQuestionPayload);
      toast.success("Question updated");
      setEditingQuestion(null);
      fetchQuestions(questionOffset);
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSavingQuestion(false);
    }
  };

  const handleDeleteTest = async () => {
    if (testId === null || !test) return;
    try {
      await deleteTest(testId);
      toast.success(`Test "${test.title}" deactivated`);
      navigate(`${prefix}/all-tests`);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  if (testId === null) {
    return (
      <DashboardLayout title="Test Not Found">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Invalid test ID in URL.</p>
        </div>
      </DashboardLayout>
    );
  }

  if (!test) {
    return (
      <DashboardLayout title="Test Detail">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout title={test.title}>
      {/* Back button + test info */}
      <div className="flex flex-col gap-4">
        <Button
          variant="ghost"
          size="sm"
          className="w-fit"
          onClick={() => navigate(`${prefix}/all-tests`)}
        >
          <ArrowLeftIcon className="size-4 mr-2" /> Back to All Tests
        </Button>

        {editingTest ? null : (
          <div className="flex items-center gap-4 flex-wrap">
            <h2 className="text-lg font-semibold">{test.title}</h2>
            <Badge variant="outline">ID: {test.test_id}</Badge>
            <Badge variant="secondary">Duration: {test.duration} min</Badge>
            <Badge variant="secondary">Subject: {test.subject_name || `#${test.subject_id}`}</Badge>
            {isAdmin && (
              <Badge variant="secondary">Coach: {test.coach_name || `#${test.coach_id}`}</Badge>
            )}
            {test.exam_date && (
              <Badge variant="secondary">Exam: {formatDateDDMMYYYY(test.exam_date)}</Badge>
            )}
            {isAdmin && (
              <>
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
                        Are you sure you want to deactivate{" "}
                        <span className="font-semibold">{test.title}</span>?
                        This test will be deactivated. Students who attempted it will keep their data.
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
              </>
            )}
            {!isAdmin && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => navigate(`${prefix}/tests/${testId}/questions`)}
              >
                View Questions
              </Button>
            )}
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
                      <TableHead className="w-28">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {assignments.map((a) => (
                      <TableRow
                        key={a.id}
                        className="cursor-pointer hover:bg-muted/50"
                        onClick={() => navigate(`${prefix}/students/${a.student_id}/assignments/${a.id}`)}
                      >
                        <TableCell className="font-mono text-sm text-muted-foreground">{a.id}</TableCell>
                        <TableCell className="font-medium">{a.student_name}</TableCell>
                        <TableCell>
                          <Badge variant="outline" className="font-mono">{a.student_code}</Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={a.status === "assigned" ? "secondary" : "default"}>{a.status}</Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground text-sm">{a.assigned_at}</TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="gap-1"
                            onClick={(e) => {
                              e.stopPropagation();
                              toast.info("SQI details coming soon");
                            }}
                          >
                            <BarChartIcon className="size-3.5" /> SQI Score
                          </Button>
                        </TableCell>
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
                onClick={() => navigate(`${prefix}/tests`)}
              >
                <PlusIcon className="size-4 mr-1" /> Add Questions
              </Button>
            </div>
          ) : (
            <>
              {isAdmin ? (
                <div className="flex flex-col gap-3">
                  {questions.map((q, idx) => (
                    <QuestionCard
                      key={q.id}
                      index={questionOffset + idx + 1}
                      question={q}
                      onEdit={() => openEditDialog(q)}
                    />
                  ))}
                </div>
              ) : (
                <div className="rounded-lg border overflow-hidden">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-12">#</TableHead>
                        <TableHead>Question</TableHead>
                        <TableHead>A</TableHead>
                        <TableHead>B</TableHead>
                        <TableHead>C</TableHead>
                        <TableHead>D</TableHead>
                        <TableHead>Answer</TableHead>
                        <TableHead>Marks</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {questions.map((q, idx) => (
                        <TableRow key={q.id}>
                          <TableCell className="font-mono text-sm text-muted-foreground">
                            {questionOffset + idx + 1}
                          </TableCell>
                          <TableCell className="font-medium max-w-[300px] truncate">
                            {q.question_text}
                          </TableCell>
                          <TableCell className="text-sm">{q.option_a}</TableCell>
                          <TableCell className="text-sm">{q.option_b}</TableCell>
                          <TableCell className="text-sm">{q.option_c}</TableCell>
                          <TableCell className="text-sm">{q.option_d}</TableCell>
                          <TableCell>
                            <Badge variant="secondary">{q.correct_answer}</Badge>
                          </TableCell>
                          <TableCell className="text-sm">{q.marks}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}

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

      {/* Edit Question Dialog (admin only) */}
      {isAdmin && (
        <Dialog open={editingQuestion !== null} onOpenChange={(open) => { if (!open) setEditingQuestion(null); }}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                Edit Question {editingQuestion ? `Q${questions.findIndex((qq) => qq.id === editingQuestion!.id) + 1 + questionOffset}` : ""}
              </DialogTitle>
              <DialogDescription>Update the question fields below.</DialogDescription>
            </DialogHeader>
            <div className="flex flex-col gap-4">
              <QuestionFormFields
                q={questionForm as QuestionDraft}
                onChange={(field, value) =>
                  setQuestionForm((prev) => ({ ...prev, [field]: value }))
                }
              />
              <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={() => setEditingQuestion(null)}>
                  Cancel
                </Button>
                <Button onClick={handleSaveQuestion} disabled={savingQuestion}>
                  <SaveIcon className="size-3 mr-1" />
                  {savingQuestion ? "Saving\u2026" : "Save"}
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Edit Test Dialog (admin only) */}
      {isAdmin && (
        <EditTestDialog
          test={editingTest ? test : null}
          open={editingTest}
          onOpenChange={(open) => {
            if (!open) {
              setEditingTest(false);
              getTest(testId).then(setTest).catch(() => {});
            }
          }}
        />
      )}
    </DashboardLayout>
  );
}
