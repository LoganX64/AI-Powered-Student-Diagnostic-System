import { createBrowserRouter } from "react-router-dom";
import App from "../src/App.tsx";
import { LoginForm } from "../src/components/login-form";
import { SignupForm } from "../src/components/signup-form";

const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: [
      { index: true, Component: LoginForm },
      { path: "signup", Component: SignupForm },
    ],
  },
]);

export default router;
