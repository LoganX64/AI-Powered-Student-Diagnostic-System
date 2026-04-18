import { LoginForm } from "../../components/login-form";
import { SignupForm } from "../../components/signup-form";
import { Button } from "../../components/ui/button";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login, register } from "../../services/auth.service.ts";

export default function AuthPage() {
  const [isLogin, setIsLogin] = useState(true);
  const [loading, setLoading] = useState(false);

  const navigate = useNavigate();

  //   login
  const handleLogin = async (data) => {
    try {
      setLoading(true);
      const res = await login(data);

      localStorage.setItem("user", JSON.stringify(res));
      navigate("/dashboard");
    } catch (error) {
      console.log((error as Error).message);
    } finally {
      setLoading(false);
    }
  };

  // signup
  const handleSignup = async (data) => {
    if (data.password !== data.confirmPassword) {
      alert("Passwords do not match");
      return;
    }
    try {
      setLoading(true);
      await register(data);
      alert("Account created. Please login.");
      setIsLogin(true);
    } catch (error) {
      console.log((error as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen">
      {isLogin ? (
        <>
          <LoginForm onSubmit={handleLogin} loading={loading} />
          <Button onClick={() => setIsLogin(false)}>Create account</Button>
        </>
      ) : (
        <>
          <SignupForm onSubmit={handleSignup} loading={loading} />
          <Button onClick={() => setIsLogin(true)}>
            Already have an account? Login
          </Button>
        </>
      )}
    </div>
  );
}
