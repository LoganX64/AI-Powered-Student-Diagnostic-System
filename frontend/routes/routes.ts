import { createBrowserRouter } from "react-router-dom";
import App from "../src/App.tsx";
import { StudentLoginPage } from "../src/features/student/StudentLoginPage.tsx";
import { StudentInstructionsPage } from "../src/features/student/StudentInstructionsPage.tsx";
import { StudentQuizPage } from "../src/features/student/StudentQuizPage.tsx";
import { StudentSubmittedPage } from "../src/features/student/StudentSubmittedPage.tsx";
import { AdminSigninPage } from "../src/features/admin/AdminSigninPage";
import { AdminSignupForm } from "../src/components/admin/signup-form.tsx";
import { AdminDashboardPage } from "../src/features/admin/AdminDashboardPage.tsx";
import { CoachSigninPage } from "../src/features/coach/CoachSigninPage.tsx";
import { CoachSignupPage } from "../src/features/coach/CoachSignupPage.tsx";
import { CoachDashboardPage } from "../src/features/coach/CoachDashboardPage.tsx";

const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: [
      { index: true, Component: StudentLoginPage },
      // Admin auth
      { path: "admin-signin", Component: AdminSigninPage },
      { path: "admin-signup", Component: AdminSignupForm },
      // Coach auth
      { path: "coach-signin", Component: CoachSigninPage },
      { path: "coach-signup", Component: CoachSignupPage },
    ],
  },

  // Full-width pages — outside the narrow App shell
  { path: "admin/dashboard", Component: AdminDashboardPage },
  { path: "coach/dashboard", Component: CoachDashboardPage },
  { path: "instructions", Component: StudentInstructionsPage },
  { path: "quiz", Component: StudentQuizPage },
  { path: "submitted", Component: StudentSubmittedPage },
]);

export default router;
