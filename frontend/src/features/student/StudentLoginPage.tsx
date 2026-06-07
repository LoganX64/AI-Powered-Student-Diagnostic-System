import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { StudentLoginForm } from "../../components/student/student-login-form";
import { loginStudent } from "../../services/student.service";
import type { StudentLoginPayload } from "../../services/student.service";

export function StudentLoginPage() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleLogin = async (data: StudentLoginPayload) => {
    try {
      setLoading(true);
      const result = await loginStudent(data);
      localStorage.setItem("student_token", result.access_token);
      localStorage.setItem("student_code", data.student_code);
      navigate("/instructions");
    } catch (error) {
      alert((error as Error).message || "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return <StudentLoginForm onSubmit={handleLogin} loading={loading} />;
}
