import { createBrowserRouter } from "react-router-dom";
import App from "../src/App.tsx";
import { StudentLoginForm } from "../src/components/student/student-login-form.tsx";
import { AdminSignupForm } from "../src/components/admin/signup-form.tsx";

const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: [
      { index: true, Component: StudentLoginForm },
      { path: "signup", Component: AdminSignupForm },
    ],
  },
]);

export default router;
