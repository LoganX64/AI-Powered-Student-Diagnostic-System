// ─── Create Payloads ──────────────────────────────────────────────────────────

export type CreateCoachPayload = {
  email: string;
  password: string;
  name: string;
};

export type CreateStudentPayload = {
  name: string;
  student_code: string;
  coach_id: number;
};

export type CreateSubjectPayload = {
  name: string;
};

export type CreateTestPayload = {
  title: string;
  subject_id: number;
  coach_id: number;
  duration: number;
  exam_date?: string;
};

export type CreateQuestionPayload = {
  question_text: string;
  option_a: string;
  option_b: string;
  option_c: string;
  option_d: string;
  correct_answer: "A" | "B" | "C" | "D";
  marks: number;
  neg_marks: number;
  importance: string;
  difficulty: string;
  type: string;
  expected_time: number;
  concept_tag: string;
};

export type CreateAssignmentPayload = {
  student_id: number;
  test_id: number;
  coach_id: number;
};

// ─── Row Types ────────────────────────────────────────────────────────────────

export type Test = {
  test_id: number;
  title: string;
  subject_id: number;
  coach_id: number;
  duration: number;
  subject_name: string;
  coach_name: string;
  exam_date?: string;
};

export type Coach = {
  coach_id: number;
  user_id: number;
  name: string;
  email: string;
  deleted_at?: string | null;
};

export type Student = {
  student_id: number;
  name: string;
  student_code: string;
  coach_id: number;
  deleted_at?: string | null;
};

export type Subject = {
  subject_id: number;
  name: string;
};

export type Assignment = {
  id: number;
  student_id: number;
  student_name: string;
  student_code: string;
  test_id: number;
  test_title: string;
  coach_id: number;
  status: string;
  assigned_at: string;
};

export type StudentDetail = {
  student_id: number;
  name: string;
  student_code: string;
  coach_id: number;
  coach_name: string;
  created_at: string;
  deleted_at: string | null;
  deleted_by_name: string | null;
  deleted_by_email: string | null;
  deleted_by_role: string | null;
};

export type StudentAssignment = {
  id: number;
  test_id: number;
  test_title: string;
  status: string;
  assigned_at: string;
  submitted: boolean;
};

export type CoachDetail = {
  coach_id: number;
  user_id: number;
  name: string;
  email: string;
  created_at: string;
};

export type CoachTest = {
  test_id: number;
  title: string;
  subject_id: number;
  duration: number;
  subject_name: string;
  exam_date?: string;
  created_at: string;
};

export type CoachStudent = {
  student_id: number;
  name: string;
  student_code: string;
  created_at: string;
};

// ─── Pagination ───────────────────────────────────────────────────────────────

export type PaginatedResponse<T> = {
  total: number;
  limit: number;
  offset: number;
  data: T[];
};

export type PaginationParams = {
  limit?: number;
  offset?: number;
  search?: string;
};

// ─── Assignment Detail ──────────────────────────────────────────────────────

export type AnswerDetail = {
  question_id: number;
  question_text: string;
  option_a: string;
  option_b: string;
  option_c: string;
  option_d: string;
  correct_answer: string;
  selected_answer: string;
  is_correct: boolean;
  marks: number;
  time_spent: number;
  marked_for_review: boolean;
  changed_answer: boolean;
  seen: boolean;
  concept_tag: string;
  difficulty: string;
};

export type AssignmentDetail = {
  student: { id: number; name: string; student_code: string };
  test: { id: number; title: string };
  assignment: { id: number; status: string; assigned_at: string };
  attempt: { id: number; submitted_at: string | null } | null;
  sqi_score: number;
  answers: AnswerDetail[];
};

// ─── SQI Types ───────────────────────────────────────────────────────────────

export type SQIAttempt = {
  attempt_id: number;
  test_id: number;
  sqi_score: number;
  analysis?: Record<string, unknown>;
};

export type SQIResponse = {
  student_id: number;
  name: string;
  attempts: SQIAttempt[];
  average_sqi: number;
  total_tests: number;
};
