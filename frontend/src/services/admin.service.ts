import { apiFetch } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────────────────────

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

// ─── Types (rows returned by list endpoints) ──────────────────────────────────

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
};

export type Student = {
  student_id: number;
  name: string;
  student_code: string;
  coach_id: number;
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

// ─── API Calls ────────────────────────────────────────────────────────────────

export const createCoach = (data: CreateCoachPayload) =>
  apiFetch<{ coach_id: number; user_id: number }>("/admin/register-coach", {
    method: "POST",
    body: JSON.stringify(data),
  });

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

export const getCoach = (coachId: number) =>
  apiFetch<CoachDetail>(`/admin/coaches/${coachId}`);

export const getCoachTests = (coachId: number, params?: PaginationParams) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  const qs = query.toString();
  return apiFetch<PaginatedResponse<CoachTest>>(`/admin/coaches/${coachId}/tests${qs ? `?${qs}` : ""}`);
};

export const getCoachStudents = (coachId: number, params?: PaginationParams) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  const qs = query.toString();
  return apiFetch<PaginatedResponse<CoachStudent>>(`/admin/coaches/${coachId}/students${qs ? `?${qs}` : ""}`);
};

export const deleteCoach = (coachId: number) =>
  apiFetch<{ message: string }>(`/admin/coaches/${coachId}`, {
    method: "DELETE",
  });

export const createStudent = (data: CreateStudentPayload) =>
  apiFetch<{ student_id: number }>("/admin/students", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const deleteStudent = (studentId: number) =>
  apiFetch<{ message: string }>(`/admin/students/${studentId}`, {
    method: "DELETE",
  });

export const getStudent = (studentId: number) =>
  apiFetch<StudentDetail>(`/admin/students/${studentId}`);

export const getStudentAssignments = (studentId: number) =>
  apiFetch<{ total: number; limit: number; offset: number; data: StudentAssignment[] }>(
    `/admin/students/${studentId}/assignments`
  );

export const createSubject = (data: CreateSubjectPayload) =>
  apiFetch<{ subject_id: number }>("/admin/subjects", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const deleteSubject = (subjectId: number) =>
  apiFetch<{ message: string }>(`/admin/subjects/${subjectId}`, {
    method: "DELETE",
  });

export const createTest = (data: CreateTestPayload) =>
  apiFetch<{ test_id: number }>("/admin/tests", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const updateTest = (testId: number, data: CreateTestPayload) =>
  apiFetch<{ message: string }>(`/admin/tests/${testId}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const deleteTest = (testId: number) =>
  apiFetch<{ message: string }>(`/admin/tests/${testId}`, {
    method: "DELETE",
  });

export const createQuestions = (testId: number, data: CreateQuestionPayload[]) =>
  apiFetch<{ question_ids: number[]; count: number }>(`/admin/tests/${testId}/questions`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const updateQuestion = (testId: number, questionId: number, data: CreateQuestionPayload) =>
  apiFetch<{ message: string }>(`/admin/tests/${testId}/questions/${questionId}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const deleteQuestion = (testId: number, questionId: number) =>
  apiFetch<{ message: string }>(`/admin/tests/${testId}/questions/${questionId}`, {
    method: "DELETE",
  });

export const createAssignment = (data: CreateAssignmentPayload) =>
  apiFetch<{ assignment_id: number }>("/admin/assignments", {
    method: "POST",
    body: JSON.stringify(data),
  });

// ─── Pagination types ─────────────────────────────────────────────────────────

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

// ─── List endpoints ────────────────────────────────────────────────────────────

export const getTests = (params?: PaginationParams) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.search) query.set("search", params.search);
  const qs = query.toString();
  return apiFetch<PaginatedResponse<Test>>(`/admin/tests${qs ? `?${qs}` : ""}`);
};

export const getStudents = (params?: PaginationParams & { include_deactivated?: boolean }) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.include_deactivated) query.set("include_deactivated", "true");
  const qs = query.toString();
  return apiFetch<PaginatedResponse<Student>>(`/admin/students${qs ? `?${qs}` : ""}`);
};

export const getCoaches = (params?: PaginationParams) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.search) query.set("search", params.search);
  const qs = query.toString();
  return apiFetch<PaginatedResponse<Coach>>(`/admin/coaches${qs ? `?${qs}` : ""}`);
};

export const getSubjects = (params?: PaginationParams) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.search) query.set("search", params.search);
  const qs = query.toString();
  return apiFetch<PaginatedResponse<Subject>>(`/admin/subjects${qs ? `?${qs}` : ""}`);
};

export const getAssignments = (params?: PaginationParams & { test_id?: number }) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.test_id) query.set("test_id", params.test_id.toString());
  const qs = query.toString();
  return apiFetch<PaginatedResponse<Assignment>>(`/admin/assignments${qs ? `?${qs}` : ""}`);
};
