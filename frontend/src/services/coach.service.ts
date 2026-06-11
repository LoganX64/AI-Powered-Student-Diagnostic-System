import { apiFetch } from "@/lib/api";
import type {
  CreateStudentPayload,
  CreateTestPayload,
  CreateQuestionPayload,
  CreateAssignmentPayload,
  CreateSubjectPayload,
  PaginatedResponse,
  PaginationParams,
  Test,
  Student,
  Subject,
} from "./admin.service";

// ─── API Calls ────────────────────────────────────────────────────────────────

export const createStudent = (data: CreateStudentPayload) =>
  apiFetch<{ student_id: number }>("/coach/students", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const createSubject = (data: CreateSubjectPayload) =>
  apiFetch<{ subject_id: number }>("/coach/subjects", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const createTest = (data: CreateTestPayload) =>
  apiFetch<{ test_id: number }>("/coach/tests", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const createQuestions = (testId: number, data: CreateQuestionPayload[]) =>
  apiFetch<{ question_ids: number[]; count: number }>(`/coach/tests/${testId}/questions`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const createAssignment = (data: CreateAssignmentPayload) =>
  apiFetch<{ assignment_id: number }>("/coach/assignments", {
    method: "POST",
    body: JSON.stringify(data),
  });

// ─── List endpoints ────────────────────────────────────────────────────────────

export const getTests = (params?: PaginationParams) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  const qs = query.toString();
  return apiFetch<PaginatedResponse<Test>>(`/coach/tests${qs ? `?${qs}` : ""}`);
};

export const getStudents = (params?: PaginationParams) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  const qs = query.toString();
  return apiFetch<PaginatedResponse<Student>>(`/coach/students${qs ? `?${qs}` : ""}`);
};

export const getSubjects = (params?: PaginationParams) => {
  const query = new URLSearchParams();
  if (params?.limit) query.set("limit", params.limit.toString());
  if (params?.offset) query.set("offset", params.offset.toString());
  const qs = query.toString();
  return apiFetch<PaginatedResponse<Subject>>(`/coach/subjects${qs ? `?${qs}` : ""}`);
};
