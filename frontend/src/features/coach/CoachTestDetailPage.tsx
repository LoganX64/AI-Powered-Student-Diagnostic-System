import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeftIcon, ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { CoachSidebar } from "@/components/coach/sidebar";
import { SiteHeader } from "@/components/admin/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiFetch } from "@/lib/api";
import { getAssignments, type Assignment, type PaginatedResponse } from "@/services/coach.service";

type TestDetail = {
  test_id: number;
  title: string;
  subject_id: number;
  coach_id: number;
  duration: number;
  created_at: string;
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

export function CoachTestDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [test, setTest] = useState<TestDetail | null>(null);
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [assignmentTotal, setAssignmentTotal] = useState(0);
  const [assignmentOffset, setAssignmentOffset] = useState(0);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [questionTotal, setQuestionTotal] = useState(0);
  const [questionOffset, setQuestionOffset] = useState(0);

  useEffect(() => {
    if (!id) return;
    apiFetch<TestDetail>(`/coach/tests/${id}`).then(setTest).catch(() => {});
  }, [id]);

  const fetchAssignments = useCallback(async (off: number) => {
    if (!id) return;
    try {
      const res = await getAssignments({ limit: PAGE_SIZE, offset: off, test_id: Number(id) });
      setAssignments(res.data);
      setAssignmentTotal(res.total);
    } catch {
      // silently ignore
    }
  }, [id]);

  const fetchQuestions = useCallback(async (off: number) => {
    if (!id) return;
    try {
      const res = await apiFetch<PaginatedResponse<Question>>(`/coach/tests/${id}/questions?limit=${PAGE_SIZE}&offset=${off}`);
      setQuestions(res.data);
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

  if (!test) {
    return (
      <SidebarProvider>
        <CoachSidebar />
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
      <CoachSidebar />
      <SidebarInset>
        <SiteHeader title={test.title} />
        <div className="flex flex-1 flex-col gap-6 p-4 lg:p-6">

          {/* Back button + test info */}
          <div className="flex flex-col gap-4">
            <Button
              variant="ghost"
              size="sm"
              className="w-fit"
              onClick={() => navigate("/coach/tests")}
            >
              <ArrowLeftIcon className="size-4 mr-2" /> Back to All Tests
            </Button>

            <div className="flex items-center gap-4 flex-wrap">
              <h2 className="text-lg font-semibold">{test.title}</h2>
              <Badge variant="outline">ID: {test.test_id}</Badge>
              <Badge variant="secondary">Duration: {test.duration}s</Badge>
              <Badge variant="secondary">Subject: {test.subject_id}</Badge>
              {test.exam_date && (
                <Badge variant="secondary">Exam: {test.exam_date}</Badge>
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={() => navigate(`/coach/tests/${test.test_id}/questions`)}
              >
                View Questions
              </Button>
            </div>
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
                <div className="flex h-32 items-center justify-center rounded-lg border border-dashed">
                  <p className="text-sm text-muted-foreground">No questions in this test.</p>
                </div>
              ) : (
                <>
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
                          <TableHead className="w-20">Difficulty</TableHead>
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
                            <TableCell>
                              <Badge variant={
                                q.difficulty === "E" ? "secondary" :
                                q.difficulty === "M" ? "outline" : "destructive"
                              }>
                                {q.difficulty === "E" ? "Easy" : q.difficulty === "M" ? "Medium" : "Hard"}
                              </Badge>
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
