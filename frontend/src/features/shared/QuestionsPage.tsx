import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  ArrowLeftIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  SaveIcon,
} from "lucide-react";
import { toast } from "sonner";
import { useRole } from "@/hooks/useRole";
import { DashboardLayout } from "@/components/shared/DashboardLayout";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
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
import { QuestionCard } from "@/components/admin/QuestionCard";
import {
  QuestionFormFields,
  type QuestionDraft,
} from "@/components/admin/forms/QuestionFormFields";
import { CreateQuestionsForm } from "@/components/admin/forms/CreateQuestionsForm";
import {
  deleteQuestion,
  updateQuestion,
  type CreateQuestionPayload,
  type PaginatedResponse,
} from "@/services/dashboard.service";
import { apiFetch } from "@/lib/api";
import { formatDateDDMMYYYY } from "@/lib/utils";

type TestDetail = {
  test_id: number;
  title: string;
  subject_id: number;
  coach_id: number;
  duration: number;
  created_at: string;
  subject_name: string;
  coach_name: string;
  exam_date?: string;
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

export function QuestionsPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const role = useRole();
  const prefix = role === "admin" ? "/admin" : "/coach";
  const isAdmin = role === "admin";
  const testId = Number(id);
  const apiPrefix = isAdmin ? "/admin" : "/coach";

  const [test, setTest] = useState<TestDetail | null>(null);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [questionTotal, setQuestionTotal] = useState(0);
  const [questionOffset, setQuestionOffset] = useState(0);

  const [editingQuestion, setEditingQuestion] = useState<Question | null>(null);
  const [questionForm, setQuestionForm] = useState<Partial<CreateQuestionPayload>>({});
  const [savingQuestion, setSavingQuestion] = useState(false);

  const [deletingQuestionId, setDeletingQuestionId] = useState<number | null>(null);
  const [testNotFound, setTestNotFound] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    apiFetch<TestDetail>(`${apiPrefix}/tests/${id}`)
      .then(setTest)
      .catch(() => setTestNotFound(true));
  }, [id, apiPrefix]);

  const fetchQuestions = useCallback(async (off: number) => {
    if (!id) return;
    try {
      const res = await apiFetch<PaginatedResponse<Question>>(
        `${apiPrefix}/tests/${id}/questions?limit=${PAGE_SIZE}&offset=${off}`
      );
      setQuestions(res.data ?? []);
      setQuestionTotal(res.total);
      setFetchError(null);
    } catch (err) {
      const message = (err as Error).message || "Failed to load questions";
      setFetchError(message);
      toast.error(message);
    }
  }, [id, apiPrefix]);

  useEffect(() => {
    fetchQuestions(questionOffset);
  }, [questionOffset, fetchQuestions]);

  const openEditDialog = (q: Question) => {
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
      importance: q.importance,
      difficulty: q.difficulty,
      type: q.type,
      expected_time: q.expected_time,
      concept_tag: q.concept_tag,
    });
  };

  const handleSaveQuestion = async () => {
    if (!editingQuestion) return;
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

  const handleDeleteQuestion = async (questionId: number) => {
    try {
      await deleteQuestion(testId, questionId);
      toast.success("Question deleted");
      setDeletingQuestionId(null);
      fetchQuestions(questionOffset);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  if (testNotFound) {
    return (
      <DashboardLayout title="Test Not Found">
        <div className="flex flex-1 flex-col items-center justify-center gap-4 p-6">
          <p className="text-muted-foreground">This test does not exist or has been deleted.</p>
          <Button variant="outline" onClick={() => navigate(`${prefix}/all-tests`)}>
            <ArrowLeftIcon className="size-4 mr-2" /> Back to All Tests
          </Button>
        </div>
      </DashboardLayout>
    );
  }

  if (!test) {
    return (
      <DashboardLayout title="Questions">
        <div className="flex flex-1 items-center justify-center p-6">
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout title={`${test.title} — Questions`}>
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

        <div className="flex items-center gap-4 flex-wrap">
          <h2 className="text-lg font-semibold">{test.title}</h2>
          <Badge variant="outline">ID: {test.test_id}</Badge>
          <Badge variant="secondary">Duration: {test.duration} min</Badge>
          {test.exam_date && (
            <Badge variant="secondary">Exam: {formatDateDDMMYYYY(test.exam_date)}</Badge>
          )}
          <Badge variant="secondary">Subject: {test.subject_name || `#${test.subject_id}`}</Badge>
        </div>
      </div>

      {/* Questions */}
      {fetchError ? (
        <div className="flex flex-col gap-4">
          <div className="flex h-24 items-center justify-center rounded-lg border border-destructive/50 bg-destructive/10">
            <p className="text-sm text-destructive">{fetchError}</p>
          </div>
          <Button variant="outline" size="sm" onClick={() => fetchQuestions(questionOffset)}>
            Retry
          </Button>
        </div>
      ) : questions.length === 0 ? (
        <div className="flex flex-col gap-4">
          <div className="flex h-24 items-center justify-center rounded-lg border border-dashed">
            <p className="text-sm text-muted-foreground">No questions in this test. Add questions below.</p>
          </div>
          <CreateQuestionsForm
            testId={testId}
            onCreated={() => fetchQuestions(0)}
          />
        </div>
      ) : (
        <>
          {isAdmin ? (
            <div className="flex flex-col gap-3">
              {questions.map((q, idx) => (
                <div key={q.id}>
                  <QuestionCard
                    index={questionOffset + idx + 1}
                    question={q}
                    onEdit={() => openEditDialog(q)}
                    onDelete={() => setDeletingQuestionId(q.id)}
                  />

                  {/* Delete confirmation dialog */}
                  <AlertDialog
                    open={deletingQuestionId === q.id}
                    onOpenChange={(open) => {
                      if (!open) setDeletingQuestionId(null);
                    }}
                  >
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete Question</AlertDialogTitle>
                        <AlertDialogDescription>
                          Are you sure you want to delete question{" "}
                          <span className="font-semibold">
                            Q{questionOffset + idx + 1}
                          </span>
                          ? This action cannot be undone.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() => handleDeleteQuestion(q.id)}
                          className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                          Delete
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
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
                    <TableHead>Difficulty</TableHead>
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
                      <TableCell className="text-sm">{q.difficulty}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          {questionTotal > PAGE_SIZE && (
            <div className="flex items-center justify-between pt-2">
              <p className="text-sm text-muted-foreground">
                Showing {questionOffset + 1}–
                {Math.min(questionOffset + PAGE_SIZE, questionTotal)} of{" "}
                {questionTotal}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={questionOffset === 0}
                  onClick={() =>
                    setQuestionOffset((o) => Math.max(0, o - PAGE_SIZE))
                  }
                >
                  <ChevronLeftIcon className="size-4" /> Prev
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={questionOffset + PAGE_SIZE >= questionTotal}
                  onClick={() =>
                    setQuestionOffset((o) => o + PAGE_SIZE)
                  }
                >
                  Next <ChevronRightIcon className="size-4" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}

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
    </DashboardLayout>
  );
}
