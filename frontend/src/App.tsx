import "./App.css";
import { LoginForm } from "./components/login-form";

function App() {
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-md">
        <LoginForm />
      </div>
    </div>
  );
}

export default App;
