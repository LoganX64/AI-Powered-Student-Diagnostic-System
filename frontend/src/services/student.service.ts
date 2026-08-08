import { apiFetch } from "@/lib/api";

// Re-export login types from auth.service for backward compatibility
export type { StudentLoginPayload, StudentLoginResponse } from "./auth.service";
export { loginStudent } from "./auth.service";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type Assignment = {
  id: number;
  test_id: number;
  test_title: string;
  status: string;
  assigned_at: string;
};

export type QuestionFromAPI = {
  id: number;
  question_text: string;
  option_a: string;
  option_b: string;
  option_c: string;
  option_d: string;
  marks: number;
  neg_marks: number;
  difficulty: string;
  type: string;
  expected_time: number;
  concept_tag: string;
};

export type AssignmentQuestionsResponse = {
  assignment_id: number;
  test_title: string;
  duration: number;
  exam_date: string;
  questions: QuestionFromAPI[];
};

export type AnswerPayload = {
  question_id: number;
  seen: boolean;
  selected_answer: string;
  time_spent: number;
  marked_for_review: boolean;
  revisited: boolean;
  changed_answer: boolean;
};

export type SubmitResponse = {
  attempt_id: number;
  total_time_spent: number;
  test_duration: number;
};

// ---------------------------------------------------------------------------
// API calls
// ---------------------------------------------------------------------------

export async function getStudentAssignments(): Promise<Assignment[]> {
  const res = await apiFetch<{ total: number; data: Assignment[] }>(
    "/student/assignments",
    {},
    "student_token"
  );
  return res.data ?? [];
}

export async function getAssignmentQuestions(
  assignmentId: number,
): Promise<AssignmentQuestionsResponse> {
  return apiFetch<AssignmentQuestionsResponse>(
    `/student/assignments/${assignmentId}/questions`,
    {},
    "student_token"
  );
}

export async function submitAnswers(
  assignmentId: number,
  answers: AnswerPayload[],
): Promise<SubmitResponse> {
  return apiFetch<SubmitResponse>(
    `/student/submit/${assignmentId}`,
    {
      method: "POST",
      body: JSON.stringify({ answers }),
    },
    "student_token"
  );
}
