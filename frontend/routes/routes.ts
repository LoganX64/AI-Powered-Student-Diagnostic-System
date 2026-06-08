import { createBrowserRouter } from "react-router-dom";
import App from "../src/App.tsx";
import { StudentLoginPage } from "../src/features/student/StudentLoginPage.tsx";
import { StudentInstructionsPage } from "../src/features/student/StudentInstructionsPage.tsx";
import { StudentQuizPage } from "../src/features/student/StudentQuizPage.tsx";
import { StudentSubmittedPage } from "../src/features/student/StudentSubmittedPage.tsx";
import { AdminSigninPage } from "../src/features/admin/AdminSigninPage";
import { AdminSignupForm } from "../src/components/admin/signup-form.tsx";
import { AdminDashboardPage } from "../src/features/admin/AdminDashboardPage.tsx";
import { CoachesPage } from "../src/features/admin/CoachesPage.tsx";
import { StudentsPage } from "../src/features/admin/StudentsPage.tsx";
import { SubjectsPage } from "../src/features/admin/SubjectsPage.tsx";
import { TestsPage } from "../src/features/admin/TestsPage.tsx";
import { CoachSigninPage } from "../src/features/coach/CoachSigninPage.tsx";
import { CoachDashboardPage } from "../src/features/coach/CoachDashboardPage.tsx";
import { CoachStudentsPage } from "../src/features/coach/CoachStudentsPage.tsx";
import { CoachSubjectsPage } from "../src/features/coach/CoachSubjectsPage.tsx";
import { CoachTestsPage } from "../src/features/coach/CoachTestsPage.tsx";

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
    ],
  },

  // Full-width pages — outside the narrow App shell
  { path: "admin/dashboard", Component: AdminDashboardPage },
  { path: "admin/coaches", Component: CoachesPage },
  { path: "admin/students", Component: StudentsPage },
  { path: "admin/subjects", Component: SubjectsPage },
  { path: "admin/tests", Component: TestsPage },
  { path: "coach/dashboard", Component: CoachDashboardPage },
  { path: "coach/students", Component: CoachStudentsPage },
  { path: "coach/subjects", Component: CoachSubjectsPage },
  { path: "coach/tests", Component: CoachTestsPage },
  { path: "instructions", Component: StudentInstructionsPage },
  { path: "quiz", Component: StudentQuizPage },
  { path: "submitted", Component: StudentSubmittedPage },
]);

export default router;
