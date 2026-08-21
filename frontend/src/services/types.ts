// ─── Create Payloads (derived from Zod schemas) ──────────────────────────────

import { z } from "zod";
import {
  createCoachSchema,
  createStudentSchema,
  updateStudentSchema,
  createSubjectSchema,
  createTestSchema,
  createQuestionSchema,
  createAssignmentSchema,
  createBatchAssignmentSchema,
} from "@/lib/validations";

export type CreateCoachPayload = z.infer<typeof createCoachSchema>;
export type CreateStudentPayload = z.infer<typeof createStudentSchema>;
export type UpdateStudentPayload = z.infer<typeof updateStudentSchema>;
export type CreateSubjectPayload = z.infer<typeof createSubjectSchema>;
export type CreateTestPayload = z.infer<typeof createTestSchema>;
export type CreateQuestionPayload = z.infer<typeof createQuestionSchema>;
export type CreateAssignmentPayload = z.infer<typeof createAssignmentSchema>;

export type IntegrityPolicy = {
  server_timing: boolean;
  autosave: boolean;
  video_proctoring: boolean;
  tab_switch_detect: boolean;
};

export type CreateBatchAssignmentPayload = z.infer<typeof createBatchAssignmentSchema>;

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
  created_at?: string;
};

export type Coach = {
  coach_id: number;
  user_id: number;
  name: string;
  email: string;
  created_at?: string;
  deleted_at?: string | null;
  subjects: CoachSubject[];
};

export type CoachStatMetric = {
  coach_id: number;
  student_count: number;
  avg_sqi: number;
};

export type Student = {
  student_id: number;
  name: string;
  student_code: string;
  coach_id: number;
  batch_id?: number | null;
  deleted_at?: string | null;
};

export type Subject = {
  subject_id: number;
  name: string;
};

export type CoachSubject = {
  subject_id: number;
  subject_name: string;
};

export type Batch = {
  id: number;
  name: string;
  created_at: string;
  student_count: number;
};

export type Assignment = {
  id: number;
  student_id: number;
  student_name: string;
  student_code: string;
  test_id: number;
  test_title: string;
  coach_id: number;
  coach_name: string;
  status: string;
  assigned_at: string;
  subject_name: string;
};

export type StudentDetail = {
  student_id: number;
  name: string;
  student_code: string;
  coach_id: number;
  batch_id: number | null;
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
  deleted_at?: string | null;
  deleted_by_name?: string | null;
  deleted_by_email?: string | null;
  deleted_by_role?: string | null;
  subjects: Subject[];
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
  sqi_score: number | null;
  analysis?: SQIAnalysis;
  answers: AnswerDetail[];
};

// ─── SQI Types ───────────────────────────────────────────────────────────────

export type SQIAnalysis = {
  version: string;
  overall_sqi: number;
  dimensions: {
    mastery: number;
    speed: number;
    risk: number;
    coverage: number;
  };
  exam_summary: {
    exam_type: string;
    has_negative_marking: boolean;
    total_questions: number;
    attempted: number;
    correct: number;
    wrong: number;
    skipped: number;
    unseen: number;
    total_marks_earned: number;
    total_marks_lost: number;
    net_score: number;
    max_possible_score: number;
    score_percent: number;
  };
  attempt_profile: {
    guessed_wrong: number;
    carefully_wrong: number;
    guessed_right: number;
    carefully_right: number;
    seen_abandoned: number;
    never_reached: number;
    neg_marks_from_guess: number;
    neg_marks_from_careful: number;
  };
  concept_profiles: Array<{
    concept_tag: string;
    subject: string;
    status: string;
    priority_rank: number;
    evidence: {
      total_questions: number;
      attempted: number;
      correct: number;
      wrong: number;
      skipped: number;
      unseen: number;
      accuracy_pct: number;
      avg_time_ratio: number;
      neg_marks_cost: number;
      guess_count: number;
      genuine_wrong: number;
      changed_to_correct: number;
      changed_to_wrong: number;
      mastery_score: number;
      priority_score: number;
    };
  }>;
  behavior_flags: {
    [key: string]: {
      detected: boolean;
      confidence: number;
      evidence: string;
    };
  };
  first_half_accuracy: number;
  second_half_accuracy: number;
};

export type SQIAttempt = {
  attempt_id: number;
  test_id: number;
  sqi_score: number;
  analysis?: SQIAnalysis;
};

export type SQIResponse = {
  student_id: number;
  name: string;
  attempts: SQIAttempt[];
  average_sqi: number | null;
  total_tests: number;
};

export type StudentSQIMetric = {
  student_id: number;
  average_sqi: number;
  total_tests: number;
};

// ─── Test Detail Types ────────────────────────────────────────────────────────

export type TestDetail = {
  test_id: number;
  title: string;
  subject_id: number;
  coach_id: number;
  duration: number;
  created_at: string;
  subject_name: string;
  coach_name: string;
  exam_date?: string;
};

export type TestQuestion = {
  id: number;
  question_text: string;
  option_a: string;
  option_b: string;
  option_c: string;
  option_d: string;
  correct_answer: string;
  marks: number;
  neg_marks: number;
  importance: string;
  difficulty: string;
  type: string;
  expected_time: number;
  concept_tag: string;
};
