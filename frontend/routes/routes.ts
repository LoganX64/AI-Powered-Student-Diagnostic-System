import { createBrowserRouter } from "react-router-dom";
import App from "../src/App.tsx";
import { StudentLoginPage } from "../src/features/student/StudentLoginPage.tsx";
import { StudentInstructionsPage } from "../src/features/student/StudentInstructionsPage.tsx";
import { StudentQuizPage } from "../src/features/student/StudentQuizPage.tsx";
import { StudentSubmittedPage } from "../src/features/student/StudentSubmittedPage.tsx";
import { AdminSigninPage } from "../src/features/admin/AdminSigninPage";
import { AdminSignupForm } from "../src/components/admin/signup-form.tsx";
import { CoachLoginForm } from "../src/components/coach/login-form.tsx";
import { AdminDashboardPage } from "../src/features/admin/AdminDashboardPage.tsx";

const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: [
      { index: true, Component: StudentLoginPage },
      { path: "admin-signin", Component: AdminSigninPage },
      { path: "admin-signup", Component: AdminSignupForm },
      { path: "coach-signin", Component: CoachLoginForm },
    ],
  },

  // Full-width pages — outside the narrow App shell
  { path: "admin/dashboard", Component: AdminDashboardPage },
  { path: "instructions", Component: StudentInstructionsPage },
  { path: "quiz", Component: StudentQuizPage },
  { path: "submitted", Component: StudentSubmittedPage },
]);

export default router;
