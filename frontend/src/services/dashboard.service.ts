import { apiFetch } from "@/lib/api";
import { getActiveRole } from "@/lib/token";
import type {
  CreateCoachPayload,
  CreateStudentPayload,
  UpdateStudentPayload,
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
  Batch,
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

import type { CreateBatchAssignmentPayload } from "./types";

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
  Batch,
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

export type { IntegrityPolicy, CreateBatchAssignmentPayload } from "./types";

// ─── Helpers ──────────────────────────────────────────────────────────────────

function getPrefix(): string {
  const role = getActiveRole();
  if (role === "coach") return "/coach";
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

function buildListQuery(
  params?: PaginationParams & {
    include_deactivated?: boolean;
    batch_id?: number;
  },
): string {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.include_deactivated) query.set("include_deactivated", "true");
  if (params?.batch_id != null)
    query.set("batch_id", params.batch_id.toString());
  if (params?.search) query.set("search", params.search);
  const qs = query.toString();
  return qs ? `?${qs}` : "";
}

function buildAssignmentQuery(
  params?: PaginationParams & {
    test_id?: number;
    status?: string;
    search?: string;
    year?: string;
    subject_id?: number;
    coach_id?: number;
  },
): string {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  if (params?.test_id) query.set("test_id", params.test_id.toString());
  if (params?.status) query.set("status", params.status);
  if (params?.search) query.set("search", params.search);
  if (params?.year) query.set("year", params.year);
  if (params?.subject_id) query.set("subject_id", params.subject_id.toString());
  if (params?.coach_id) query.set("coach_id", params.coach_id.toString());
  const qs = query.toString();
  return qs ? `?${qs}` : "";
}

// ─── Coach endpoints (admin-only) ─────────────────────────────────────────────

export const createCoach = (data: CreateCoachPayload) =>
  apiFetch<{ coach_id: number; user_id: number }>("/admin/register-coach", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const getCoaches = (
  params?: PaginationParams & { include_deactivated?: boolean },
) =>
  apiFetch<PaginatedResponse<Coach>>(`/admin/coaches${buildListQuery(params)}`);

export const getCoachStatsBatch = (coachIds: number[]) =>
  apiFetch<{ data: CoachStatMetric[] }>("/admin/coaches/stats-batch", {
    method: "POST",
    body: JSON.stringify({ coach_ids: coachIds }),
  });

export const getCoach = (coachId: number) =>
  apiFetch<CoachDetail>(`/admin/coaches/${coachId}`);

export const getCoachTests = (coachId: number, params?: PaginationParams) =>
  apiFetch<PaginatedResponse<CoachTest>>(
    `/admin/coaches/${coachId}/tests${buildQuery(params)}`,
  );

export const getCoachStudents = (coachId: number, params?: PaginationParams) =>
  apiFetch<PaginatedResponse<CoachStudent>>(
    `/admin/coaches/${coachId}/students${buildQuery(params)}`,
  );

export const deleteCoach = (coachId: number) =>
  apiFetch<{ message: string }>(`/admin/coaches/${coachId}`, {
    method: "DELETE",
  });

export const reactivateCoach = (coachId: number) =>
  apiFetch<{ message: string }>(`/admin/coaches/${coachId}/reactivate`, {
    method: "PUT",
  });

export const updateCoach = (
  coachId: number,
  data: { name: string; email: string; subject_ids: number[] },
) =>
  apiFetch<{ message: string }>(`/admin/coaches/${coachId}`, {
    method: "PUT",
    body: JSON.stringify(data),
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
  apiFetch<{ message: string }>(
    `${getPrefix()}/students/${studentId}/reactivate`,
    {
      method: "PUT",
    },
  );

export const updateStudent = (studentId: number, data: UpdateStudentPayload) =>
  apiFetch<{ message: string }>(`${getPrefix()}/students/${studentId}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const getStudent = (studentId: number) =>
  apiFetch<StudentDetail>(`${getPrefix()}/students/${studentId}`);

export const getStudentAssignments = (
  studentId: number,
  params?: PaginationParams & { status?: string },
) =>
  apiFetch<{
    total: number;
    limit: number;
    offset: number;
    data: StudentAssignment[];
  }>(
    `${getPrefix()}/students/${studentId}/assignments${buildAssignmentQuery(params)}`,
  );

export const getStudents = (
  params?: PaginationParams & {
    include_deactivated?: boolean;
    batch_id?: number;
  },
) =>
  apiFetch<PaginatedResponse<Student>>(
    `${getPrefix()}/students${buildListQuery(params)}`,
  );

// ─── Batch endpoints (role-aware, tenant-wide) ────────────────────────────────

export const getBatches = () =>
  apiFetch<{ data: Batch[] }>(`${getPrefix()}/batches`);

export const createBatch = (name: string) =>
  apiFetch<{ batch_id: number }>(`${getPrefix()}/batches`, {
    method: "POST",
    body: JSON.stringify({ name }),
  });

export const deleteBatch = (batchId: number) =>
  apiFetch<{ message: string; students_reassigned: number }>(
    `${getPrefix()}/batches/${batchId}`,
    { method: "DELETE" },
  );

export const updateBatch = (batchId: number, name: string) =>
  apiFetch<{ message: string }>(`${getPrefix()}/batches/${batchId}`, {
    method: "PUT",
    body: JSON.stringify({ name }),
  });

export const transferStudentBatch = (
  studentId: number,
  batchId: number | null,
) =>
  apiFetch<{ message: string }>(`${getPrefix()}/students/${studentId}/batch`, {
    method: "PATCH",
    body: JSON.stringify({ batch_id: batchId }),
  });

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
  apiFetch<{ message: string }>(
    `${getPrefix()}/subjects/${subjectId}/reactivate`,
    {
      method: "PUT",
    },
  );

export const updateSubject = (subjectId: number, name: string) =>
  apiFetch<{ message: string }>(`${getPrefix()}/subjects/${subjectId}`, {
    method: "PUT",
    body: JSON.stringify({ name }),
  });

export const getSubjects = (params?: PaginationParams) =>
  apiFetch<PaginatedResponse<Subject>>(
    `${getPrefix()}/subjects${buildQuery(params)}`,
  );

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
  apiFetch<PaginatedResponse<Test>>(
    `${getPrefix()}/tests${buildQuery(params)}`,
  );

export const getTest = (testId: number) =>
  apiFetch<TestDetail>(`${getPrefix()}/tests/${testId}`);

export const getTestQuestions = (testId: number, params?: PaginationParams) =>
  apiFetch<PaginatedResponse<TestQuestion>>(
    `${getPrefix()}/tests/${testId}/questions${buildQuery(params)}`,
  );

// ─── Question endpoints (role-aware) ──────────────────────────────────────────

export const createQuestions = (
  testId: number,
  data: CreateQuestionPayload[],
) =>
  apiFetch<{ question_ids: number[]; count: number }>(
    `${getPrefix()}/tests/${testId}/questions`,
    {
      method: "POST",
      body: JSON.stringify(data),
    },
  );

export const updateQuestion = (
  testId: number,
  questionId: number,
  data: CreateQuestionPayload,
) =>
  apiFetch<{ message: string }>(
    `${getPrefix()}/tests/${testId}/questions/${questionId}`,
    {
      method: "PUT",
      body: JSON.stringify(data),
    },
  );

export const deleteQuestion = (testId: number, questionId: number) =>
  apiFetch<{ message: string }>(
    `${getPrefix()}/tests/${testId}/questions/${questionId}`,
    {
      method: "DELETE",
    },
  );

// ─── Assignment endpoints (role-aware) ────────────────────────────────────────

export const createAssignment = (data: CreateAssignmentPayload) =>
  apiFetch<{ assignment_id: number }>(`${getPrefix()}/assignments`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const createBatchAssignment = (data: CreateBatchAssignmentPayload) =>
  apiFetch<{ created: number }>(`${getPrefix()}/assignments/batch`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const getAssignments = (
  params?: PaginationParams & {
    test_id?: number;
    status?: string;
    year?: string;
    subject_id?: number;
    coach_id?: number;
  },
) =>
  apiFetch<PaginatedResponse<Assignment>>(
    `${getPrefix()}/assignments${buildAssignmentQuery(params)}`,
  );

export const deleteAssignment = (assignmentId: number) =>
  apiFetch<{ message: string }>(`${getPrefix()}/assignments/${assignmentId}`, {
    method: "DELETE",
  });

export const deleteVideo = (assignmentId: number) =>
  apiFetch<{ message: string }>(
    `${getPrefix()}/assignments/${assignmentId}/video`,
    {
      method: "DELETE",
    },
  );

export const getVideoToken = (assignmentId: number) =>
  apiFetch<{ token: string; expires_in: number; assignment_id: number }>(
    `${getPrefix()}/assignments/${assignmentId}/video-token`,
    { method: "POST" },
  );

// ─── Assignment Detail endpoint (role-aware) ───────────────────────────────

export const getAssignmentDetail = (studentId: number, assignmentId: number) =>
  apiFetch<AssignmentDetail>(
    `${getPrefix()}/students/${studentId}/assignments/${assignmentId}`,
  );

// ─── SQI endpoint (role-aware) ────────────────────────────────────────────

export const getStudentSQI = (
  studentId: number,
  opts?: { compute?: boolean; include_analysis?: boolean },
) => {
  const query = new URLSearchParams();
  if (opts?.compute) query.set("compute", "true");
  if (opts?.include_analysis) query.set("include_analysis", "true");
  const qs = query.toString();
  return apiFetch<SQIResponse>(
    `${getPrefix()}/students/${studentId}/sqi${qs ? `?${qs}` : ""}`,
  );
};

export const getStudentSQIBatch = (studentIds: number[]) =>
  apiFetch<{ data: StudentSQIMetric[] }>(`${getPrefix()}/students/sqi-batch`, {
    method: "POST",
    body: JSON.stringify({ student_ids: studentIds }),
  });

// ─── Scalable SQI compute + jobs (role-aware) ─────────────────────────────

export type Job = {
  id: number;
  tenant_id: number;
  type: string;
  total: number;
  done: number;
  failed: number;
  status: string;
  created_at: string;
  updated_at: string;
};

export type ComputeResponse = { job_id: number; total: number };

export const computeSQI = (attemptId: number) =>
  apiFetch<ComputeResponse>(`${getPrefix()}/sqi/compute`, {
    method: "POST",
    body: JSON.stringify({ attempt_id: attemptId }),
  });

export const computeSQIBatch = (testId: number) =>
  apiFetch<ComputeResponse>(`${getPrefix()}/sqi/compute-batch`, {
    method: "POST",
    body: JSON.stringify({ test_id: testId }),
  });

export const getJob = (jobId: number) =>
  apiFetch<Job>(`${getPrefix()}/jobs/${jobId}`);

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
