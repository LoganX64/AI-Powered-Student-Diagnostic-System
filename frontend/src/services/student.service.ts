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

export type IntegrityPolicy = {
  server_timing: boolean;
  autosave: boolean;
  video_proctoring: boolean;
  tab_switch_detect: boolean;
};

export type AssignmentQuestionsResponse = {
  assignment_id: number;
  test_title: string;
  duration: number;
  exam_date: string;
  questions: QuestionFromAPI[];
  integrity_policy?: IntegrityPolicy | null;
};

export type AutosaveAnswer = {
  question_id: number;
  selected_answer: string;
  seen: boolean;
  time_spent: number;
  marked_for_review: boolean;
  revisited: boolean;
  changed_answer: boolean;
};

export type StartExamResponse = {
  attempt_id: number;
  deadline: string;
  server_now: string;
};

export type ExamStateResponse = {
  attempt_id: number;
  deadline: string;
  remaining_seconds: number;
  answers: Array<{
    question_id: number;
    selected_answer: string;
    time_spent: number;
    marked_for_review: boolean;
    seen: boolean;
    revisited: boolean;
    changed_answer: boolean;
  }>;
};

export type AutosaveResponse = { saved: number };

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

export async function startExam(
  assignmentId: number,
): Promise<StartExamResponse> {
  return apiFetch<StartExamResponse>(
    `/student/assignments/${assignmentId}/start`,
    { method: "POST" },
    "student_token",
  );
}

export async function autosaveAnswers(
  assignmentId: number,
  answers: AutosaveAnswer[],
): Promise<AutosaveResponse> {
  return apiFetch<AutosaveResponse>(
    `/student/assignments/${assignmentId}/autosave`,
    {
      method: "POST",
      body: JSON.stringify({ answers }),
    },
    "student_token",
  );
}

export async function getExamState(
  assignmentId: number,
): Promise<ExamStateResponse> {
  return apiFetch<ExamStateResponse>(
    `/student/assignments/${assignmentId}/state`,
    {},
    "student_token",
  );
}

export async function uploadVideoChunk(
  assignmentId: number,
  index: number,
  blob: Blob,
): Promise<{ received_index: string }> {
  const form = new FormData();
  form.append("index", String(index));
  form.append("chunk", blob, `${index}.webm`);
  return apiFetch<{ received_index: string }>(
    `/student/assignments/${assignmentId}/video-chunk`,
    { method: "POST", body: form },
    "student_token",
  );
}

export async function submitExam(
  assignmentId: number,
  answers: AnswerPayload[],
): Promise<SubmitResponse> {
  return apiFetch<SubmitResponse>(
    `/student/assignments/${assignmentId}/submit`,
    {
      method: "POST",
      body: JSON.stringify({ answers }),
    },
    "student_token"
  );
}
