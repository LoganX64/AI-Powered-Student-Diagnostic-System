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

// ─── API Calls ────────────────────────────────────────────────────────────────

export const createCoach = (data: CreateCoachPayload) =>
  apiFetch<{ coach_id: number; user_id: number }>("/admin/register-coach", {
    method: "POST",
    body: JSON.stringify(data),
  });

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

export const createQuestions = (testId: number, data: CreateQuestionPayload[]) =>
  apiFetch<{ question_ids: number[]; count: number }>(`/admin/tests/${testId}/questions`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const createAssignment = (data: CreateAssignmentPayload) =>
  apiFetch<{ assignment_id: number }>("/admin/assignments", {
    method: "POST",
    body: JSON.stringify(data),
  });

// ─── List endpoints ────────────────────────────────────────────────────────────

export const getTests = () =>
  apiFetch<Test[]>("/admin/tests");

export const getStudents = () =>
  apiFetch<Student[]>("/admin/students");

export const getCoaches = () =>
  apiFetch<Coach[]>("/admin/coaches");

export const getSubjects = () =>
  apiFetch<Subject[]>("/admin/subjects");
