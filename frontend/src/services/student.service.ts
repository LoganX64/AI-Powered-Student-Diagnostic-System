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
// API calls
// ---------------------------------------------------------------------------

export async function loginStudent(
  data: StudentLoginPayload,
): Promise<StudentLoginResponse> {
  const response = await fetch(`${BASE_URL}/student/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
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

export async function submitAnswers(
  assignmentId: number,
  answers: AnswerPayload[],
): Promise<SubmitResponse> {
  const token = localStorage.getItem("student_token");

  const response = await fetch(`${BASE_URL}/student/submit/${assignmentId}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
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
