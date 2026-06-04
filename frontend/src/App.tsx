import "./App.css";
import { Outlet } from "react-router-dom";

function App() {
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-md">
        <Outlet />
      </div>
    </div>
  );
}

export default App;
