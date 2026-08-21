import { z } from "zod";

export function zodErrors(err: z.ZodError): Record<string, string> {
  const out: Record<string, string> = {};
  err.issues.forEach((issue) => {
    out[issue.path.join(".")] = issue.message;
  });
  return out;
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

export const loginSchema = z.object({
  email: z.string().min(1, "Email is required").email("Invalid email"),
  password: z.string().min(1, "Password is required"),
});

export const adminSignupSchema = z
  .object({
    orgName: z.string().min(1, "Organization name is required"),
    email: z.string().min(1, "Email is required").email("Invalid email"),
    password: z
      .string()
      .min(8, "Password must be at least 8 characters"),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  });

export const studentLoginSchema = z.object({
  student_code: z.string().min(1, "Student code is required"),
});

// ─── CRUD ─────────────────────────────────────────────────────────────────────

export const createCoachSchema = z.object({
  name: z.string().min(1, "Name is required"),
  email: z.string().min(1, "Email is required").email("Invalid email"),
  password: z.string().min(8, "Password must be at least 8 characters"),
  subject_ids: z.array(z.number().int().positive()).min(1, "At least one subject is required"),
});

export const createStudentSchema = z.object({
  name: z.string().min(1, "Name is required"),
  student_code: z.string().optional(),
  coach_id: z.number().int().positive("Please select a coach").optional(),
  batch_id: z.number().int().nullable().optional(),
});

export const updateStudentSchema = z.object({
  name: z.string().min(1, "Name is required"),
  student_code: z.string().optional(),
  coach_id: z.number().int().positive("Please select a coach").optional(),
  batch_id: z.number().int().nullable().optional(),
});

export const createSubjectSchema = z.object({
  name: z.string().min(1, "Subject name is required"),
});

export const createTestSchema = z.object({
  title: z.string().min(1, "Title is required").max(200),
  subject_id: z.number().int().positive("Please select a subject"),
  subject_name: z.string().min(1),
  coach_id: z.number().int().min(0),
  duration: z
    .number()
    .int()
    .min(1, "Duration must be at least 1 minute")
    .max(600),
  exam_date: z.string().optional(),
});

export const createQuestionSchema = z.object({
  question_text: z.string().min(1, "Question text is required").max(1000),
  option_a: z.string().min(1, "Option A is required").max(500),
  option_b: z.string().min(1, "Option B is required").max(500),
  option_c: z.string().min(1, "Option C is required").max(500),
  option_d: z.string().min(1, "Option D is required").max(500),
  correct_answer: z.enum(["A", "B", "C", "D"]),
  marks: z.number().min(0.25, "Minimum 0.25 marks").max(100),
  neg_marks: z.number().min(0).max(100),
  importance: z.enum(["high", "medium", "low"]),
  difficulty: z.enum(["E", "M", "H"]),
  type: z.enum(["mcq", "multi", "integer"]),
  expected_time: z.number().min(0).max(300),
  concept_tag: z.string().max(100).optional().or(z.literal("")),
});

export const createQuestionsBatchSchema = z.object({
  test_id: z.number().int().positive("Valid Test ID required"),
  questions: z
    .array(createQuestionSchema)
    .min(1, "At least one question required"),
});

const integrityPolicySchema = z.object({
  server_timing: z.boolean(),
  autosave: z.boolean(),
  video_proctoring: z.boolean(),
  tab_switch_detect: z.boolean(),
});

export const createAssignmentSchema = z.object({
  student_id: z.number().int().positive("Please select a student"),
  test_id: z.number().int().positive("Please select a test"),
  coach_id: z.number().int(),
  integrity_policy: integrityPolicySchema.optional(),
  estimated_cost: z.number().optional(),
});

export const createBatchAssignmentSchema = z.object({
  test_id: z.number().int().positive("Please select a test"),
  student_ids: z.array(z.number().int()).min(1, "Select at least one student").optional(),
  batch_ids: z.array(z.number().int()).min(1, "Select at least one batch").optional(),
  coach_id: z.number().int(),
  integrity_policy: integrityPolicySchema.optional(),
  estimated_cost: z.number().optional(),
});

// ─── Settings / Support ───────────────────────────────────────────────────────

export const contactSupportSchema = z.object({
  subject: z.string().min(1, "Subject is required"),
  message: z.string().min(1, "Message is required"),
});

export const profileSettingsSchema = z.object({
  name: z.string().min(1, "Name is required"),
  email: z.string().min(1, "Email is required").email("Invalid email"),
  phone: z.string().optional(),
});

export const changePasswordSchema = z
  .object({
    currentPassword: z.string().min(1, "Current password is required"),
    newPassword: z
      .string()
      .min(8, "New password must be at least 8 characters"),
    confirmNewPassword: z.string(),
  })
  .refine((data) => data.newPassword === data.confirmNewPassword, {
    message: "Passwords do not match",
    path: ["confirmNewPassword"],
  });

export const notificationPreferencesSchema = z.object({
  emailNotifications: z.boolean(),
  pushNotifications: z.boolean(),
  weeklyDigest: z.boolean(),
  testAlerts: z.boolean(),
  studentActivity: z.boolean(),
});
