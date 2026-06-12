import { createBrowserRouter } from "react-router-dom";
import { ProtectedRoute } from "../src/components/ProtectedRoute.tsx";
import { LandingPage } from "../src/features/landing/LandingPage.tsx";
import { AboutPage } from "../src/features/landing/AboutPage.tsx";
import { StudentLoginPage } from "../src/features/student/StudentLoginPage.tsx";
import { StudentInstructionsPage } from "../src/features/student/StudentInstructionsPage.tsx";
import { StudentQuizPage } from "../src/features/student/StudentQuizPage.tsx";
import { StudentSubmittedPage } from "../src/features/student/StudentSubmittedPage.tsx";
import { AdminSigninPage } from "../src/features/admin/AdminSigninPage.tsx";
import { AdminSignupPage } from "../src/features/admin/AdminSignupPage.tsx";
import { AdminDashboardPage } from "../src/features/admin/AdminDashboardPage.tsx";
import { CoachesPage } from "../src/features/admin/CoachesPage.tsx";
import { CoachDetailPage } from "../src/features/admin/CoachDetailPage.tsx";
import { StudentsPage } from "../src/features/admin/StudentsPage.tsx";
import { SubjectsPage } from "../src/features/admin/SubjectsPage.tsx";
import { TestsPage } from "../src/features/admin/TestsPage.tsx";
import { AllTestsPage } from "../src/features/admin/AllTestsPage.tsx";
import { TestDetailPage } from "../src/features/admin/TestDetailPage.tsx";
import { QuestionsPage } from "../src/features/admin/QuestionsPage.tsx";
import { StudentDetailPage } from "../src/features/admin/StudentDetailPage.tsx";
import { StudentSQIPage } from "../src/features/admin/StudentSQIPage.tsx";
import { CoachSigninPage } from "../src/features/coach/CoachSigninPage.tsx";
import { CoachDashboardPage } from "../src/features/coach/CoachDashboardPage.tsx";
import { CoachStudentsPage } from "../src/features/coach/CoachStudentsPage.tsx";
import { CoachSubjectsPage } from "../src/features/coach/CoachSubjectsPage.tsx";
import { CoachTestsPage } from "../src/features/coach/CoachTestsPage.tsx";
import { CoachAllTestsPage } from "../src/features/coach/CoachAllTestsPage.tsx";
import { CoachTestDetailPage } from "../src/features/coach/CoachTestDetailPage.tsx";
import { CoachQuestionsPage } from "../src/features/coach/CoachQuestionsPage.tsx";
import { CoachStudentDetailPage } from "../src/features/coach/CoachStudentDetailPage.tsx";
import { CoachStudentSQIPage } from "../src/features/coach/CoachStudentSQIPage.tsx";

const router = createBrowserRouter([
  // Public pages
  { path: "/", Component: LandingPage },
  { path: "about", Component: AboutPage },

  // Auth pages — centered layout
  { path: "student-login", Component: StudentLoginPage },
  { path: "admin-signin", Component: AdminSigninPage },
  { path: "admin-signup", Component: AdminSignupPage },
  { path: "coach-signin", Component: CoachSigninPage },

  // Protected routes — require authentication
  {
    Component: ProtectedRoute,
    children: [
      // Admin dashboard
      { path: "admin/dashboard", Component: AdminDashboardPage },
      { path: "admin/coaches", Component: CoachesPage },
      { path: "admin/coaches/:id", Component: CoachDetailPage },
      { path: "admin/students", Component: StudentsPage },
      { path: "admin/students/:id", Component: StudentDetailPage },
      { path: "admin/students/:id/sqi", Component: StudentSQIPage },
      { path: "admin/subjects", Component: SubjectsPage },
      { path: "admin/tests", Component: TestsPage },
      { path: "admin/all-tests", Component: AllTestsPage },
      { path: "admin/tests/:id", Component: TestDetailPage },
      { path: "admin/tests/:id/questions", Component: QuestionsPage },

      // Coach dashboard
      { path: "coach/dashboard", Component: CoachDashboardPage },
      { path: "coach/students", Component: CoachStudentsPage },
      { path: "coach/students/:id", Component: CoachStudentDetailPage },
      { path: "coach/students/:id/sqi", Component: CoachStudentSQIPage },
      { path: "coach/subjects", Component: CoachSubjectsPage },
      { path: "coach/tests", Component: CoachTestsPage },
      { path: "coach/all-tests", Component: CoachAllTestsPage },
      { path: "coach/tests/:id", Component: CoachTestDetailPage },
      { path: "coach/tests/:id/questions", Component: CoachQuestionsPage },

      // Student flow
      { path: "instructions", Component: StudentInstructionsPage },
      { path: "quiz", Component: StudentQuizPage },
      { path: "submitted", Component: StudentSubmittedPage },
    ],
  },
]);

export default router;
