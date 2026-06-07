export type StudentLoginPayload = {
  student_code: string;
};

export type StudentLoginResponse = {
  access_token: string;
};

export async function loginStudent(
  data: StudentLoginPayload,
): Promise<StudentLoginResponse> {
  const response = await fetch("/student/login", {
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
