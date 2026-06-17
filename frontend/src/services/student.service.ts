const BASE_URL = import.meta.env.VITE_BACKEND_URL;

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type StudentLoginPayload = {
  student_code: string;
};

export type StudentLoginResponse = {
  access_token: string;
};

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
  was_initially_wrong: boolean;
};

export type SubmitResponse = {
  attempt_id: number;
  sqi_score: number;
  total_time_spent: number;
  test_duration: number;
  analysis: Record<string, unknown>;
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function authHeaders(): Record<string, string> {
  const token = localStorage.getItem("student_token");
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// ---------------------------------------------------------------------------
// API calls
// ---------------------------------------------------------------------------

export async function loginStudent(
  data: StudentLoginPayload,
): Promise<StudentLoginResponse> {
  const response = await fetch(`${BASE_URL}/student/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  type ErrorResponse = { error: string };
  const payload = (await response
    .json()
    .catch(() => ({ error: "Invalid response" }))) as
    | StudentLoginResponse
    | ErrorResponse;

  if (!response.ok) {
    const errorMessage =
      "error" in payload ? payload.error : "Student login failed";
    throw new Error(errorMessage);
  }

  return payload as StudentLoginResponse;
}

export async function getStudentAssignments(): Promise<Assignment[]> {
  const response = await fetch(`${BASE_URL}/student/assignments`, {
    method: "GET",
    headers: { "Content-Type": "application/json", ...authHeaders() },
  });

  type ErrorResponse = { error: string };
  type SuccessResponse = { total: number; data: Assignment[] };
  const payload = (await response
    .json()
    .catch(() => ({ error: "Invalid response" }))) as
    | SuccessResponse
    | ErrorResponse;

  if (!response.ok) {
    const errorMessage =
      "error" in payload ? payload.error : "Failed to fetch assignments";
    throw new Error(errorMessage);
  }

  return (payload as SuccessResponse).data ?? [];
}

export async function getAssignmentQuestions(
  assignmentId: number,
): Promise<AssignmentQuestionsResponse> {
  const response = await fetch(
    `${BASE_URL}/student/assignments/${assignmentId}/questions`,
    {
      method: "GET",
      headers: { "Content-Type": "application/json", ...authHeaders() },
    },
  );

  type ErrorResponse = { error: string };
  const payload = (await response
    .json()
    .catch(() => ({ error: "Invalid response" }))) as
    | AssignmentQuestionsResponse
    | ErrorResponse;

  if (!response.ok) {
    const errorMessage =
      "error" in payload ? payload.error : "Failed to fetch questions";
    throw new Error(errorMessage);
  }

  return payload as AssignmentQuestionsResponse;
}

export async function submitAnswers(
  assignmentId: number,
  answers: AnswerPayload[],
): Promise<SubmitResponse> {
  const response = await fetch(`${BASE_URL}/student/submit/${assignmentId}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify({ answers }),
  });

  type ErrorResponse = { error: string };
  const payload = (await response
    .json()
    .catch(() => ({ error: "Invalid response" }))) as
    | SubmitResponse
    | ErrorResponse;

  if (!response.ok) {
    const errorMessage =
      "error" in payload ? payload.error : "Submission failed";
    throw new Error(errorMessage);
  }

  return payload as SubmitResponse;
}
