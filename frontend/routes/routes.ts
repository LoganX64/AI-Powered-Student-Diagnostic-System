import { createBrowserRouter } from "react-router-dom";
import App from "../src/App.tsx";
import { StudentLoginForm } from "../src/components/student/student-login-form.tsx";
// import { StudentSignupForm } from "../src/components/student/student-signup-form.tsx";
import { AdminSignupForm } from "../src/components/admin/signup-form.tsx";
import { AdminLoginForm } from "../src/components/admin/login-form.tsx";
// import { CoachSignupForm } from "../src/components/coach/signup-form.tsx";
import { CoachLoginForm } from "../src/components/coach/login-form.tsx";

const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: [{ index: true, Component: StudentLoginForm }],
  },
  // {
  //   path: "/student-signup",
  //   Component: StudentSignupForm,
  // },
  {
    path: "/admin-signin",
    Component: AdminLoginForm,
  },
  {
    path: "/admin-signup",
    Component: AdminSignupForm,
  },
  {
    path: "/coach-signin",
    Component: CoachLoginForm,
    children: [{ index: true, Component: CoachLoginForm }],
  },
  // {
  //   path: "/coach-signup",
  //   Component: CoachSignupForm,
  //   children: [{ index: true, Component: CoachSignupForm }],
  // },
]);

export default router;
