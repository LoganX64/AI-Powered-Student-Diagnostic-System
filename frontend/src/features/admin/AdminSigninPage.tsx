import { LoginPage } from "../../components/shared/login-page";

export function AdminSigninPage() {
  return (
    <div className="flex min-h-dvh w-full items-center justify-center px-4 py-8 sm:py-12">
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
