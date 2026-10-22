import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { StudentLoginForm } from "../../components/student/student-login-form";
import { loginStudent, type StudentLoginPayload } from "../../services/auth.service";
import { useSession } from "@/contexts/SessionContext";

export function StudentLoginPage() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();
  const { refresh } = useSession();

  const handleLogin = async (data: StudentLoginPayload) => {
    try {
      setError("");
      setLoading(true);
      await loginStudent(data);
      localStorage.setItem("student_code", data.student_code);
      await refresh();
      navigate("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-dvh w-full items-center justify-center px-4 py-8 sm:py-12">
      <StudentLoginForm onSubmit={handleLogin} loading={loading} error={error} />
    </div>
  );
}
