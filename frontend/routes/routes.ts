import { createBrowserRouter } from "react-router-dom";
import { ProtectedRoute } from "../src/components/ProtectedRoute.tsx";
import { LandingPage } from "../src/features/landing/LandingPage.tsx";
import { AboutPage } from "../src/features/landing/AboutPage.tsx";
import { StudentLoginPage } from "../src/features/student/StudentLoginPage.tsx";
import { StudentDashboardPage } from "../src/features/student/StudentDashboardPage.tsx";
import { StudentInstructionsPage } from "../src/features/student/StudentInstructionsPage.tsx";
import { StudentQuizPage } from "../src/features/student/StudentQuizPage.tsx";
import { StudentSubmittedPage } from "../src/features/student/StudentSubmittedPage.tsx";
import { AdminSigninPage } from "../src/features/admin/AdminSigninPage.tsx";
import { AdminSignupPage } from "../src/features/admin/AdminSignupPage.tsx";
import { CoachSigninPage } from "../src/features/coach/CoachSigninPage.tsx";

  // Unified pages
  import { DashboardPage } from "../src/features/shared/DashboardPage.tsx";
  import { CoachesPage } from "../src/features/shared/CoachesPage.tsx";
  import { CoachDetailPage } from "../src/features/shared/CoachDetailPage.tsx";
  import { StudentsPage } from "../src/features/shared/StudentsPage.tsx";
  import { StudentDetailPage } from "../src/features/shared/StudentDetailPage.tsx";
  import { StudentSQIPage } from "../src/features/shared/StudentSQIPage.tsx";
  import { SubjectsPage } from "../src/features/shared/SubjectsPage.tsx";
  import { TestsPage } from "../src/features/shared/TestsPage.tsx";
  import { AllTestsPage } from "../src/features/shared/AllTestsPage.tsx";
  import { TestDetailPage } from "../src/features/shared/TestDetailPage.tsx";
  import { TestDetailsPage } from "../src/features/shared/TestDetailsPage.tsx";
import { QuestionsPage } from "../src/features/shared/QuestionsPage.tsx";
import { SettingsPage } from "../src/features/shared/SettingsPage.tsx";
import { GetHelpPage } from "../src/features/shared/GetHelpPage.tsx";
import { AccountsPage } from "../src/features/shared/AccountsPage.tsx";
import { BatchesPage } from "../src/features/shared/BatchesPage.tsx";
import { BillingPage } from "../src/features/shared/BillingPage.tsx";
import { NotificationsPage } from "../src/features/shared/NotificationsPage.tsx";
import { AssignmentDetailPage } from "../src/features/shared/AssignmentDetailPage.tsx";
import { AssignTestPage } from "../src/features/shared/AssignTestPage.tsx";

const router = createBrowserRouter([
  // Public pages
  { path: "/", Component: LandingPage },
  { path: "/about", Component: AboutPage },


  // Auth pages — centered layout
  { path: "student-login", Component: StudentLoginPage },
  { path: "admin-signin", Component: AdminSigninPage },
  { path: "admin-signup", Component: AdminSignupPage },
  { path: "coach-signin", Component: CoachSigninPage },

  // Protected routes — require authentication
  {
    Component: ProtectedRoute,
    children: [
      // Admin dashboard routes
      { path: "admin/dashboard", Component: DashboardPage },
      { path: "admin/coaches", Component: CoachesPage },
      { path: "admin/coaches/:id", Component: CoachDetailPage },
      { path: "admin/students", Component: StudentsPage },
      { path: "admin/batches", Component: BatchesPage },
      { path: "admin/students/:id", Component: StudentDetailPage },
      { path: "admin/students/:id/sqi", Component: StudentSQIPage },
      { path: "admin/students/:id/assignments/:assignmentId", Component: AssignmentDetailPage },
      { path: "admin/subjects", Component: SubjectsPage },
      { path: "admin/tests", Component: TestsPage },
      { path: "admin/all-tests", Component: AllTestsPage },
      { path: "admin/assign-test", Component: AssignTestPage },
      { path: "admin/test-details", Component: TestDetailsPage },
      { path: "admin/tests/:id", Component: TestDetailPage },
      { path: "admin/tests/:id/questions", Component: QuestionsPage },
      { path: "admin/settings", Component: SettingsPage },
      { path: "admin/help", Component: GetHelpPage },
      { path: "admin/accounts", Component: AccountsPage },
      { path: "admin/billing", Component: BillingPage },
      { path: "admin/notifications", Component: NotificationsPage },

      // Coach dashboard routes
      { path: "coach/dashboard", Component: DashboardPage },
      { path: "coach/students", Component: StudentsPage },
      { path: "coach/batches", Component: BatchesPage },
      { path: "coach/students/:id", Component: StudentDetailPage },
      { path: "coach/students/:id/sqi", Component: StudentSQIPage },
      { path: "coach/students/:id/assignments/:assignmentId", Component: AssignmentDetailPage },
      { path: "coach/subjects", Component: SubjectsPage },
      { path: "coach/tests", Component: TestsPage },
      { path: "coach/all-tests", Component: AllTestsPage },
      { path: "coach/assign-test", Component: AssignTestPage },
      { path: "coach/test-details", Component: TestDetailsPage },
      { path: "coach/tests/:id", Component: TestDetailPage },
      { path: "coach/tests/:id/questions", Component: QuestionsPage },
      { path: "coach/settings", Component: SettingsPage },
      { path: "coach/help", Component: GetHelpPage },
      { path: "coach/accounts", Component: AccountsPage },
      { path: "coach/notifications", Component: NotificationsPage },

      // Student flow
      { path: "dashboard", Component: StudentDashboardPage },
      { path: "instructions", Component: StudentInstructionsPage },
      { path: "quiz", Component: StudentQuizPage },
      { path: "submitted", Component: StudentSubmittedPage },
    ],
  },
]);

export default router;
