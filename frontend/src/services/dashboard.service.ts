import { apiFetch } from "@/lib/api";
import type {
  CreateCoachPayload,
  CreateStudentPayload,
  CreateSubjectPayload,
  CreateTestPayload,
  CreateQuestionPayload,
  CreateAssignmentPayload,
  Test,
  TestDetail,
  TestQuestion,
  Coach,
  Student,
  Subject,
  Assignment,
  StudentDetail,
  StudentAssignment,
  CoachDetail,
  CoachTest,
  CoachStudent,
  PaginatedResponse,
  PaginationParams,
  AssignmentDetail,
  SQIResponse,
  StudentSQIMetric,
  CoachStatMetric,
} from "./types";

export type {
  CreateCoachPayload,
  CreateStudentPayload,
  CreateSubjectPayload,
  CreateTestPayload,
  CreateQuestionPayload,
  CreateAssignmentPayload,
  Test,
  TestDetail,
  TestQuestion,
  Coach,
  Student,
  Subject,
  Assignment,
  StudentDetail,
  StudentAssignment,
  CoachDetail,
  CoachTest,
  CoachStudent,
  PaginatedResponse,
  PaginationParams,
  AssignmentDetail,
  SQIResponse,
  StudentSQIMetric,
  CoachStatMetric,
};

// ─── Helpers ──────────────────────────────────────────────────────────────────

function getPrefix(): string {
  const adminRole = localStorage.getItem("admin_role");
  if (adminRole === "admin") return "/admin";
  const coachRole = localStorage.getItem("coach_role");
  if (coachRole === "coach") return "/coach";
  return "/admin";
}

function buildQuery(params?: PaginationParams): string {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.search) query.set("search", params.search);
  const qs = query.toString();
  return qs ? `?${qs}` : "";
}

function buildListQuery(params?: PaginationParams & { include_deactivated?: boolean }): string {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.include_deactivated) query.set("include_deactivated", "true");
  if (params?.search) query.set("search", params.search);
  const qs = query.toString();
  return qs ? `?${qs}` : "";
}

function buildAssignmentQuery(params?: PaginationParams & { test_id?: number }): string {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.test_id) query.set("test_id", params.test_id.toString());
  const qs = query.toString();
  return qs ? `?${qs}` : "";
}

// ─── Coach endpoints (admin-only) ─────────────────────────────────────────────

export const createCoach = (data: CreateCoachPayload) =>
  apiFetch<{ coach_id: number; user_id: number }>("/admin/register-coach", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const getCoaches = (params?: PaginationParams & { include_deactivated?: boolean }) =>
  apiFetch<PaginatedResponse<Coach>>(`/admin/coaches${buildListQuery(params)}`);

export const getCoachStatsBatch = (coachIds: number[]) =>
  apiFetch<{ data: CoachStatMetric[] }>("/admin/coaches/stats-batch", {
    method: "POST",
    body: JSON.stringify({ coach_ids: coachIds }),
  });

export const getCoach = (coachId: number) =>
  apiFetch<CoachDetail>(`/admin/coaches/${coachId}`);

export const getCoachTests = (coachId: number, params?: PaginationParams) =>
  apiFetch<PaginatedResponse<CoachTest>>(`/admin/coaches/${coachId}/tests${buildQuery(params)}`);

export const getCoachStudents = (coachId: number, params?: PaginationParams) =>
  apiFetch<PaginatedResponse<CoachStudent>>(`/admin/coaches/${coachId}/students${buildQuery(params)}`);

export const deleteCoach = (coachId: number) =>
  apiFetch<{ message: string }>(`/admin/coaches/${coachId}`, {
    method: "DELETE",
  });

export const reactivateCoach = (coachId: number) =>
  apiFetch<{ message: string }>(`/admin/coaches/${coachId}/reactivate`, {
    method: "PUT",
  });

// ─── Student endpoints (role-aware) ───────────────────────────────────────────

export const createStudent = (data: CreateStudentPayload) =>
  apiFetch<{ student_id: number }>(`${getPrefix()}/students`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const deleteStudent = (studentId: number) =>
  apiFetch<{ message: string }>(`${getPrefix()}/students/${studentId}`, {
    method: "DELETE",
  });

export const reactivateStudent = (studentId: number) =>
  apiFetch<{ message: string }>(`${getPrefix()}/students/${studentId}/reactivate`, {
    method: "PUT",
  });

export const getStudent = (studentId: number) =>
  apiFetch<StudentDetail>(`${getPrefix()}/students/${studentId}`);

export const getStudentAssignments = (studentId: number) =>
  apiFetch<{ total: number; limit: number; offset: number; data: StudentAssignment[] }>(
    `${getPrefix()}/students/${studentId}/assignments`
  );

export const getStudents = (params?: PaginationParams & { include_deactivated?: boolean }) =>
  apiFetch<PaginatedResponse<Student>>(`${getPrefix()}/students${buildListQuery(params)}`);

// ─── Subject endpoints (role-aware) ───────────────────────────────────────────

export const createSubject = (data: CreateSubjectPayload) =>
  apiFetch<{ subject_id: number }>(`${getPrefix()}/subjects`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const deleteSubject = (subjectId: number) =>
  apiFetch<{ message: string }>(`${getPrefix()}/subjects/${subjectId}`, {
    method: "DELETE",
  });

export const reactivateSubject = (subjectId: number) =>
  apiFetch<{ message: string }>(`${getPrefix()}/subjects/${subjectId}/reactivate`, {
    method: "PUT",
  });

export const getSubjects = (params?: PaginationParams) =>
  apiFetch<PaginatedResponse<Subject>>(`${getPrefix()}/subjects${buildQuery(params)}`);

// ─── Test endpoints (role-aware) ──────────────────────────────────────────────

export const createTest = (data: CreateTestPayload) =>
  apiFetch<{ test_id: number }>(`${getPrefix()}/tests`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const updateTest = (testId: number, data: CreateTestPayload) =>
  apiFetch<{ message: string }>(`${getPrefix()}/tests/${testId}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const deleteTest = (testId: number) =>
  apiFetch<{ message: string }>(`${getPrefix()}/tests/${testId}`, {
    method: "DELETE",
  });

export const getTests = (params?: PaginationParams) =>
  apiFetch<PaginatedResponse<Test>>(`${getPrefix()}/tests${buildQuery(params)}`);

export const getTest = (testId: number) =>
  apiFetch<TestDetail>(`${getPrefix()}/tests/${testId}`);

export const getTestQuestions = (testId: number, params?: PaginationParams) =>
  apiFetch<PaginatedResponse<TestQuestion>>(`${getPrefix()}/tests/${testId}/questions${buildQuery(params)}`);

// ─── Question endpoints (role-aware) ──────────────────────────────────────────

export const createQuestions = (testId: number, data: CreateQuestionPayload[]) =>
  apiFetch<{ question_ids: number[]; count: number }>(`${getPrefix()}/tests/${testId}/questions`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const updateQuestion = (testId: number, questionId: number, data: CreateQuestionPayload) =>
  apiFetch<{ message: string }>(`${getPrefix()}/tests/${testId}/questions/${questionId}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const deleteQuestion = (testId: number, questionId: number) =>
  apiFetch<{ message: string }>(`${getPrefix()}/tests/${testId}/questions/${questionId}`, {
    method: "DELETE",
  });

// ─── Assignment endpoints (role-aware) ────────────────────────────────────────

export const createAssignment = (data: CreateAssignmentPayload) =>
  apiFetch<{ assignment_id: number }>(`${getPrefix()}/assignments`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const getAssignments = (params?: PaginationParams & { test_id?: number }) =>
  apiFetch<PaginatedResponse<Assignment>>(`${getPrefix()}/assignments${buildAssignmentQuery(params)}`);

// ─── Assignment Detail endpoint (role-aware) ───────────────────────────────

export const getAssignmentDetail = (studentId: number, assignmentId: number) =>
  apiFetch<AssignmentDetail>(`${getPrefix()}/students/${studentId}/assignments/${assignmentId}`);

// ─── SQI endpoint (role-aware) ────────────────────────────────────────────

export const getStudentSQI = (studentId: number, opts?: { compute?: boolean; include_analysis?: boolean }) => {
  const query = new URLSearchParams();
  if (opts?.compute) query.set("compute", "true");
  if (opts?.include_analysis) query.set("include_analysis", "true");
  const qs = query.toString();
  return apiFetch<SQIResponse>(`${getPrefix()}/students/${studentId}/sqi${qs ? `?${qs}` : ""}`);
};

export const getStudentSQIBatch = (studentIds: number[]) =>
  apiFetch<{ data: StudentSQIMetric[] }>(`${getPrefix()}/students/sqi-batch`, {
    method: "POST",
    body: JSON.stringify({ student_ids: studentIds }),
  });

// ─── Dashboard stats (role-aware) ───────────────────────────────────────────

export type DashboardCounts = {
  totalCoaches: number;
  totalStudents: number;
  testsCreated: number;
};

export const getDashboardCounts = async (): Promise<DashboardCounts> => {
  const prefix = getPrefix();

  const [coaches, students, tests] = await Promise.allSettled([
    apiFetch<PaginatedResponse<unknown>>(`${prefix}/coaches?limit=1`),
    apiFetch<PaginatedResponse<unknown>>(`${prefix}/students?limit=1`),
    apiFetch<PaginatedResponse<unknown>>(`${prefix}/tests?limit=1`),
  ]);

  return {
    totalCoaches: coaches.status === "fulfilled" ? coaches.value.total : 0,
    totalStudents: students.status === "fulfilled" ? students.value.total : 0,
    testsCreated: tests.status === "fulfilled" ? tests.value.total : 0,
  };
};
