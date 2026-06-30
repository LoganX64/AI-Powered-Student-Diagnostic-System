import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { StudentLoginForm } from "../../components/student/student-login-form";
import { loginStudent, type StudentLoginPayload } from "../../services/auth.service";

export function StudentLoginPage() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const handleLogin = async (data: StudentLoginPayload) => {
    try {
      setError("");
      setLoading(true);
      const result = await loginStudent(data);
      localStorage.setItem("student_token", result.token);
      localStorage.setItem("student_code", data.student_code);
      navigate("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <StudentLoginForm onSubmit={handleLogin} loading={loading} error={error} />
    </div>
  );
}
