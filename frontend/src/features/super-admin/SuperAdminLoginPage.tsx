import { LoginPage } from "../../components/shared/login-page";

export function SuperAdminSigninPage() {
  return (
    <div className="flex min-h-dvh w-full items-center justify-center px-4 py-8 sm:py-12">
      <LoginPage
        title="Super Admin"
        description="Sign in to the platform control panel"
        dashboardPath="/super-admin/dashboard"
        footerLinks={[
          { label: "Admin?", linkText: "Login here", href: "/admin-signin" },
        ]}
      />
    </div>
  );
}
