import { LoginPage } from "../../components/shared/login-page";

export function AdminSigninPage() {
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <LoginPage
        title="Welcome back"
        description="Sign in to your admin account"
        dashboardPath="/admin/dashboard"
        footerLinks={[
          { label: "Don't have an account?", linkText: "Sign up", href: "/admin-signup" },
          { label: "Coach?", linkText: "Login here", href: "/coach-signin" },
        ]}
      />
    </div>
  );
}
