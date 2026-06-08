import { createBrowserRouter } from "react-router-dom";
import App from "../src/App.tsx";
import { StudentLoginPage } from "../src/features/student/StudentLoginPage.tsx";
import { StudentInstructionsPage } from "../src/features/student/StudentInstructionsPage.tsx";
import { StudentQuizPage } from "../src/features/student/StudentQuizPage.tsx";
import { StudentSubmittedPage } from "../src/features/student/StudentSubmittedPage.tsx";
import { AdminLoginForm } from "../src/components/admin/login-form.tsx";
import { AdminSignupForm } from "../src/components/admin/signup-form.tsx";
import { CoachLoginForm } from "../src/components/coach/login-form.tsx";

const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: [
      { index: true, Component: StudentLoginPage },

      // Admin / coach routes (use the narrow centered layout from App)
      { path: "admin-signin", Component: AdminLoginForm },
      { path: "admin-signup", Component: AdminSignupForm },
      { path: "coach-signin", Component: CoachLoginForm },
    ],
  },

  // Full-width exam routes — rendered outside the narrow App shell
  { path: "instructions", Component: StudentInstructionsPage },
  { path: "quiz", Component: StudentQuizPage },
  { path: "submitted", Component: StudentSubmittedPage },
]);

export default router;
