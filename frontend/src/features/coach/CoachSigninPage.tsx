import { LoginPage } from "../../components/shared/login-page";

export function CoachSigninPage() {
  return (
    <div className="flex min-h-dvh w-full items-center justify-center px-4 py-8 sm:py-12">
      <LoginPage
        title="Welcome back, Coach!"
        description="Sign in to your coach account"
        emailPlaceholder="coach@example.com"
        dashboardPath="/coach/dashboard"
        footerLinks={[
          { label: "Contact your admin to get access", linkText: "", href: "" },
          { label: "Admin?", linkText: "Login here", href: "/admin-signin" },
        ]}
      />
    </div>
  );
}
